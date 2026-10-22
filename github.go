package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
)

type IssueGroupType string

type IssueGroup struct {
	Query string
	Team  string
	Type  IssueGroupType
}

// issueCountKey identifies one issue_total series.
type issueCountKey struct {
	Type  IssueGroupType
	Team  string
	State string
}

// githubCollector serves issue counts from a cache. The counts come from the
// search API, which allows only 30 requests per minute, so they are refreshed
// in the background at a fixed pace (Run) instead of on every scrape.
type githubCollector struct {
	githubClient *github.Client

	issueCounter *prometheus.Desc
	lastRefresh  *prometheus.Desc

	// searchInterval is the pause before each search request.
	searchInterval time.Duration
	// maxRateLimitWait caps how long a rate-limited request waits for a retry.
	maxRateLimitWait time.Duration

	mu     sync.RWMutex
	counts map[issueCountKey]int
	// lastComplete is when the last refresh with no failed count finished.
	lastComplete time.Time
}

// errRateLimited is returned when a search is still rate limited after the
// retry.
var errRateLimited = errors.New("search still rate limited after retry")

const (
	REPO_OWNER = "giantswarm"
	REPO_NAME  = "giantswarm"

	TEAM_LABEL_PREFIX = "team/"

	BASE_QUERY = "is:issue repo:" + REPO_OWNER + "/" + REPO_NAME

	// SEARCH_INTERVAL keeps the search requests at 24 per minute, under the
	// limit of 30.
	SEARCH_INTERVAL = 2500 * time.Millisecond

	// REFRESH_INTERVAL is the time from the start of one refresh to the start
	// of the next, matching the scrape interval.
	REFRESH_INTERVAL = 5 * time.Minute

	// RETRY_INTERVAL is used instead after a refresh that was rate limited,
	// or that failed while there were no counts yet, right after a restart.
	RETRY_INTERVAL = time.Minute

	MAX_RATE_LIMIT_WAIT = 2 * time.Minute

	// DEFAULT_RETRY_AFTER is the wait when GitHub gives no retry time.
	DEFAULT_RETRY_AFTER = time.Minute
)

var (
	TEAM_LABEL_BLOCKLIST = []string{
		"null",
		"team",
		"teddyfriends",
	}

	All         IssueGroupType = "all"
	Postmortems IssueGroupType = "postmortem"

	issueStates = []string{"open", "closed"}
)

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}

	return false
}

func NewGitHubCollector(auth githubAuthConfig) (*githubCollector, error) {
	tc, err := auth.httpClient(context.Background())
	if err != nil {
		return nil, err
	}

	return newGitHubCollector(github.NewClient(tc)), nil
}

func newGitHubCollector(client *github.Client) *githubCollector {
	return &githubCollector{
		githubClient: client,

		issueCounter: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "github", "issue_total"),
			"Number of issues in GitHub",
			[]string{"type", TEAM_METRIC_LABEL, "state"},
			nil,
		),
		lastRefresh: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "github", "last_complete_refresh_timestamp_seconds"),
			"Time of the last refresh of the GitHub issue counts in which no count failed",
			nil,
			nil,
		),

		searchInterval:   SEARCH_INTERVAL,
		maxRateLimitWait: MAX_RATE_LIMIT_WAIT,

		counts: map[issueCountKey]int{},
	}
}

// Run refreshes the counts until ctx is done.
func (c *githubCollector) Run(ctx context.Context) {
	for {
		start := time.Now()
		retrySoon := c.refresh(ctx)

		wait := REFRESH_INTERVAL - time.Since(start)
		if retrySoon {
			wait = RETRY_INTERVAL
		}
		if err := sleep(ctx, wait); err != nil {
			return
		}
	}
}

// issueGroups lists the team labels and returns the groups to count. It uses
// the labels API, which is not subject to the search rate limit.
func (c *githubCollector) issueGroups(ctx context.Context) ([]IssueGroup, error) {
	issueGroups := []IssueGroup{
		{
			Query: "",
			Type:  All,
		},
		{
			Query: "label:postmortem",
			Type:  Postmortems,
		},
	}

	opt := &github.ListOptions{PerPage: 100}
	for {
		reqCtx, cancel := context.WithTimeout(ctx, githubHTTPTimeout)
		labels, resp, err := c.githubClient.Issues.ListLabels(reqCtx, REPO_OWNER, REPO_NAME, opt)
		cancel()
		if err != nil {
			return nil, err
		}

		for _, label := range labels {
			fullLabelName := label.GetName()
			labelName := strings.TrimPrefix(fullLabelName, TEAM_LABEL_PREFIX)

			if !strings.HasPrefix(fullLabelName, TEAM_LABEL_PREFIX) {
				continue
			}
			if contains(TEAM_LABEL_BLOCKLIST, labelName) {
				continue
			}
			// GitHub search has no escape for a quote inside a quoted value.
			if strings.Contains(fullLabelName, `"`) {
				log.Printf("skipping label %q: it contains a double quote", fullLabelName)
				continue
			}

			issueGroups = append(issueGroups, IssueGroup{
				Query: "label:postmortem " + "label:\"" + fullLabelName + "\"",
				Team:  labelName,
				Type:  Postmortems,
			})
		}

		if resp.NextPage == 0 {
			return issueGroups, nil
		}
		opt.Page = resp.NextPage
	}
}

// refresh recounts every group and reports whether to retry soon: when the
// team labels cannot be listed, after a persistent rate limit, or when it
// fails while there were no counts yet (after a restart).
// A count that fails keeps its last value, so one failed request does not
// drop a series. When the rate limit persists after the retry, the refresh
// stops and the counts not reached yet keep their last value too. Series of
// removed team labels are dropped.
func (c *githubCollector) refresh(ctx context.Context) bool {
	issueGroups, err := c.issueGroups(ctx)
	if err != nil {
		log.Printf("listing team labels: %v", err)
		return true
	}

	c.mu.RLock()
	previous := c.counts
	c.mu.RUnlock()

	counts := map[issueCountKey]int{}
	complete := true
	rateLimited := false
	for _, issueGroup := range issueGroups {
		for _, state := range issueStates {
			key := issueCountKey{Type: issueGroup.Type, Team: issueGroup.Team, State: state}

			if !rateLimited {
				parts := []string{BASE_QUERY, "is:" + state}
				if issueGroup.Query != "" {
					parts = append(parts, issueGroup.Query)
				}

				total, err := c.countIssues(ctx, strings.Join(parts, " "))
				if err == nil {
					counts[key] = total
					continue
				}
				if ctx.Err() != nil {
					return false
				}
				log.Print(err)
				complete = false
				rateLimited = errors.Is(err, errRateLimited)
			}

			if count, ok := previous[key]; ok {
				counts[key] = count
			}
		}
	}

	c.mu.Lock()
	c.counts = counts
	if complete {
		c.lastComplete = time.Now()
	}
	c.mu.Unlock()

	return rateLimited || (!complete && len(previous) == 0)
}

// countIssues runs one search after the search interval. When it hits the
// rate limit, it waits for the reset and tries once more.
func (c *githubCollector) countIssues(ctx context.Context, query string) (int, error) {
	if err := sleep(ctx, c.searchInterval); err != nil {
		return 0, err
	}

	total, err := c.search(ctx, query)
	wait, limited := c.rateLimitWait(err)
	if !limited {
		return total, err
	}

	log.Printf("search rate limit hit, retrying in %s", wait)
	if err := sleep(ctx, wait); err != nil {
		return 0, err
	}

	total, err = c.search(ctx, query)
	if _, limited := c.rateLimitWait(err); limited {
		return 0, errRateLimited
	}
	return total, err
}

func (c *githubCollector) search(ctx context.Context, query string) (int, error) {
	// The token mode HTTP client has no timeout of its own.
	ctx, cancel := context.WithTimeout(ctx, githubHTTPTimeout)
	defer cancel()

	result, _, err := c.githubClient.Search.Issues(ctx, query, &github.SearchOptions{
		ListOptions: github.ListOptions{Page: 1, PerPage: 1}},
	)
	if err != nil {
		return 0, err
	}

	return result.GetTotal(), nil
}

// rateLimitWait reports whether err is a rate limit error and how long to wait
// before retrying, at most maxRateLimitWait.
func (c *githubCollector) rateLimitWait(err error) (time.Duration, bool) {
	var wait time.Duration

	var rateLimitErr *github.RateLimitError
	var abuseErr *github.AbuseRateLimitError
	var errResp *github.ErrorResponse
	switch {
	case errors.As(err, &rateLimitErr):
		wait = DEFAULT_RETRY_AFTER
		if reset := rateLimitErr.Rate.Reset.Time; !reset.IsZero() {
			wait = time.Until(reset) + time.Second
		}
	case errors.As(err, &abuseErr):
		wait = DEFAULT_RETRY_AFTER
		if abuseErr.RetryAfter != nil {
			wait = *abuseErr.RetryAfter
		}
	case errors.As(err, &errResp) && isSecondaryRateLimit(errResp):
		// go-github v17 recognises only the old abuse documentation URL, so
		// a 429 or a 403 for a secondary rate limit arrives as ErrorResponse.
		wait = DEFAULT_RETRY_AFTER
		if seconds, err := strconv.Atoi(errResp.Response.Header.Get("Retry-After")); err == nil {
			wait = time.Duration(seconds) * time.Second
		}
	default:
		return 0, false
	}

	if wait < c.searchInterval {
		wait = c.searchInterval
	}
	if wait > c.maxRateLimitWait {
		wait = c.maxRateLimitWait
	}
	return wait, true
}

func isSecondaryRateLimit(errResp *github.ErrorResponse) bool {
	if errResp.Response == nil {
		return false
	}

	switch errResp.Response.StatusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusForbidden:
		return errResp.Response.Header.Get("Retry-After") != "" ||
			strings.Contains(strings.ToLower(errResp.Message), "rate limit")
	default:
		return false
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *githubCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.issueCounter
	ch <- c.lastRefresh
}

func (c *githubCollector) Collect(ch chan<- prometheus.Metric) {
	// refresh replaces the map and never changes a published one, so it is
	// read outside the lock and a slow scrape does not block the refresh.
	c.mu.RLock()
	counts := c.counts
	lastComplete := c.lastComplete
	c.mu.RUnlock()

	if !lastComplete.IsZero() {
		ch <- prometheus.MustNewConstMetric(
			c.lastRefresh,
			prometheus.GaugeValue,
			float64(lastComplete.Unix()),
		)
	}

	for key, count := range counts {
		ch <- prometheus.MustNewConstMetric(
			c.issueCounter,
			prometheus.GaugeValue,
			float64(count),
			string(key.Type), key.Team, key.State,
		)
	}
}
