package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// fakeGitHub serves the labels and search endpoints the collector uses.
type fakeGitHub struct {
	t *testing.T

	mu       sync.Mutex
	queries  []string
	totals   map[string]int
	failures map[string]int // query -> remaining failures
	limited  int            // remaining rate limit responses for any query
	tooMany  int            // remaining 429 responses for any query
	// remaining 403 responses for a secondary rate limit, for any query
	secondary int
	// dropHoneybadger removes the team/honeybadger label
	dropHoneybadger bool
	onSearch        func() // called after each search is recorded
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/repos/giantswarm/giantswarm/labels":
		// Two pages, so pagination is covered.
		if r.URL.Query().Get("page") == "2" {
			f.mu.Lock()
			drop := f.dropHoneybadger
			f.mu.Unlock()
			if drop {
				_, _ = fmt.Fprint(w, `[{"name":"team/null"}]`)
				return
			}
			_, _ = fmt.Fprint(w, `[{"name":"team/null"},{"name":"team/honeybadger"}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s?page=2>; rel="next"`, "http://"+r.Host+r.URL.Path))
		_, _ = fmt.Fprint(w, `[{"name":"bug"},{"name":"team/cabbage"},{"name":"postmortem"}]`)
	case "/search/issues":
		q := r.URL.Query().Get("q")

		f.mu.Lock()
		f.queries = append(f.queries, q)
		limited := f.limited > 0
		if limited {
			f.limited--
		}
		failing := f.failures[q] > 0
		if failing {
			f.failures[q]--
		}
		tooMany := !limited && f.tooMany > 0
		if tooMany {
			f.tooMany--
		}
		secondary := !limited && !tooMany && f.secondary > 0
		if secondary {
			f.secondary--
		}
		total := f.totals[q]
		onSearch := f.onSearch
		f.mu.Unlock()

		if onSearch != nil {
			onSearch()
		}
		if secondary {
			// The current documentation URL, which go-github v17 does not
			// recognise as an abuse rate limit.
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprint(w, `{"message":"You have exceeded a secondary rate limit. Please wait a few minutes before you try again.","documentation_url":"https://docs.github.com/rest/overview/rate-limits-for-the-rest-api#about-secondary-rate-limits"}`)
			return
		}
		if tooMany {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprint(w, `{"message":"You have exceeded a secondary rate limit."}`)
			return
		}
		if limited {
			w.Header().Set("X-RateLimit-Limit", "30")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprint(w, `{"message":"API rate limit exceeded for installation."}`)
			return
		}
		if failing {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, `{"message":"boom"}`)
			return
		}
		_, _ = fmt.Fprintf(w, `{"total_count":%d,"items":[]}`, total)
	default:
		f.t.Errorf("unexpected request %s", r.URL)
		w.WriteHeader(http.StatusNotFound)
	}
}

func newTestCollector(t *testing.T, f *fakeGitHub) *githubCollector {
	t.Helper()

	server := httptest.NewServer(f)
	t.Cleanup(server.Close)

	client := github.NewClient(nil)
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = baseURL

	c := newGitHubCollector(client)
	c.searchInterval = time.Millisecond
	c.maxRateLimitWait = 10 * time.Millisecond
	return c
}

func query(parts ...string) string {
	return strings.Join(append([]string{BASE_QUERY}, parts...), " ")
}

const issueTotal = "operations_github_issue_total"

const wantMetrics = `
# HELP operations_github_issue_total Number of issues in GitHub
# TYPE operations_github_issue_total gauge
operations_github_issue_total{state="closed",team="",type="all"} 100
operations_github_issue_total{state="closed",team="",type="postmortem"} 10
operations_github_issue_total{state="closed",team="cabbage",type="postmortem"} 4
operations_github_issue_total{state="closed",team="honeybadger",type="postmortem"} 6
operations_github_issue_total{state="open",team="",type="all"} 50
operations_github_issue_total{state="open",team="",type="postmortem"} 5
operations_github_issue_total{state="open",team="cabbage",type="postmortem"} 2
operations_github_issue_total{state="open",team="honeybadger",type="postmortem"} 3
`

func testTotals() map[string]int {
	return map[string]int{
		query("is:open"):                                                  50,
		query("is:closed"):                                                100,
		query("is:open", "label:postmortem"):                              5,
		query("is:closed", "label:postmortem"):                            10,
		query("is:open", "label:postmortem label:\"team/cabbage\""):       2,
		query("is:closed", "label:postmortem label:\"team/cabbage\""):     4,
		query("is:open", "label:postmortem label:\"team/honeybadger\""):   3,
		query("is:closed", "label:postmortem label:\"team/honeybadger\""): 6,
	}
}

func TestCollectBeforeRefreshIsEmpty(t *testing.T) {
	c := newTestCollector(t, &fakeGitHub{t: t})

	if n := testutil.CollectAndCount(c); n != 0 {
		t.Fatalf("got %d metrics before the first refresh, want 0", n)
	}
}

func TestRefresh(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)

	c.refresh(context.Background())

	if err := testutil.CollectAndCompare(c, strings.NewReader(wantMetrics), issueTotal); err != nil {
		t.Fatal(err)
	}
	// Scrapes are served from the cache, without further requests.
	if len(f.queries) != 8 {
		t.Fatalf("got %d searches, want 8", len(f.queries))
	}
	testutil.CollectAndCount(c)
	if len(f.queries) != 8 {
		t.Fatalf("collect made %d searches, want none", len(f.queries)-8)
	}
}

func TestRefreshKeepsLastValueOnError(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)
	c.refresh(context.Background())

	// Second round: one count changes, another one fails.
	f.totals[query("is:open")] = 51
	f.totals[query("is:open", "label:postmortem label:\"team/cabbage\"")] = 99
	f.failures = map[string]int{query("is:open", "label:postmortem label:\"team/cabbage\""): 1}
	c.refresh(context.Background())

	want := strings.Replace(wantMetrics, `{state="open",team="",type="all"} 50`, `{state="open",team="",type="all"} 51`, 1)
	if err := testutil.CollectAndCompare(c, strings.NewReader(want), issueTotal); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshRetriesAfterRateLimit(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals(), limited: 1}
	c := newTestCollector(t, f)

	c.refresh(context.Background())

	if err := testutil.CollectAndCompare(c, strings.NewReader(wantMetrics), issueTotal); err != nil {
		t.Fatal(err)
	}
	if len(f.queries) != 9 {
		t.Fatalf("got %d searches, want 9 (8 plus one retry)", len(f.queries))
	}
}

func TestRefreshPacesSearches(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)
	c.searchInterval = 20 * time.Millisecond

	start := time.Now()
	c.refresh(context.Background())

	if elapsed := time.Since(start); elapsed < 8*c.searchInterval {
		t.Fatalf("8 searches took %s, want at least %s", elapsed, 8*c.searchInterval)
	}
}

func TestRefreshStopsOnCancel(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)
	c.refresh(context.Background())
	f.queries = nil

	// Cancel during the second refresh, after its first search.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.onSearch = cancel
	// Long enough that the next search's wait is still running at cancel.
	c.searchInterval = 50 * time.Millisecond

	done := make(chan struct{})
	go func() {
		c.refresh(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not stop after cancel")
	}

	if len(f.queries) > 1 {
		t.Fatalf("got %d searches after cancel, want at most 1", len(f.queries))
	}
	// A cancelled refresh leaves the previous counts in place.
	if err := testutil.CollectAndCompare(c, strings.NewReader(wantMetrics), issueTotal); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshStopsOnPersistentRateLimit(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)
	c.refresh(context.Background())
	f.queries = nil
	completed := c.lastComplete

	// The first search and its retry are both rate limited.
	f.limited = 2
	f.totals[query("is:closed")] = 101
	c.refresh(context.Background())

	if len(f.queries) != 2 {
		t.Fatalf("got %d searches, want 2 (the search and its retry)", len(f.queries))
	}
	// Every count keeps its last value, none of the later ones is fetched.
	if err := testutil.CollectAndCompare(c, strings.NewReader(wantMetrics), issueTotal); err != nil {
		t.Fatal(err)
	}
	if !c.lastComplete.Equal(completed) {
		t.Fatal("an aborted refresh must not move the last complete refresh time")
	}
}

func TestRefreshRetriesAfterTooManyRequests(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals(), tooMany: 1}
	c := newTestCollector(t, f)

	c.refresh(context.Background())

	if err := testutil.CollectAndCompare(c, strings.NewReader(wantMetrics), issueTotal); err != nil {
		t.Fatal(err)
	}
	if len(f.queries) != 9 {
		t.Fatalf("got %d searches, want 9 (8 plus one retry)", len(f.queries))
	}
}

func TestLastCompleteRefresh(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)

	if n := testutil.CollectAndCount(c, "operations_github_last_complete_refresh_timestamp_seconds"); n != 0 {
		t.Fatalf("got %d timestamp series before the first refresh, want 0", n)
	}

	c.refresh(context.Background())
	if n := testutil.CollectAndCount(c, "operations_github_last_complete_refresh_timestamp_seconds"); n != 1 {
		t.Fatalf("got %d timestamp series after a refresh, want 1", n)
	}
	completed := c.lastComplete

	// A refresh with a failed count is not complete.
	f.failures = map[string]int{query("is:open"): 1}
	c.refresh(context.Background())
	if !c.lastComplete.Equal(completed) {
		t.Fatal("a refresh with a failed count must not move the last complete refresh time")
	}
}

func TestRefreshRetriesAfterSecondaryRateLimit(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals(), secondary: 1}
	c := newTestCollector(t, f)

	if c.refresh(context.Background()) {
		t.Fatal("refresh asks for a quick retry, want the secondary rate limit retried in place")
	}
	if len(f.queries) != 9 {
		t.Fatalf("got %d searches, want 9 (8 plus one retry)", len(f.queries))
	}
}

func TestRefreshDropsRemovedLabelsWhenIncomplete(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals()}
	c := newTestCollector(t, f)
	c.refresh(context.Background())

	// The honeybadger label is gone and the first search is rate limited
	// for good, so every count keeps its last value except honeybadger's.
	f.dropHoneybadger = true
	f.limited = 2
	if !c.refresh(context.Background()) {
		t.Fatal("a rate limited refresh must ask for a quick retry")
	}

	var want []string
	for _, line := range strings.Split(wantMetrics, "\n") {
		if !strings.Contains(line, "honeybadger") {
			want = append(want, line)
		}
	}
	if err := testutil.CollectAndCompare(c, strings.NewReader(strings.Join(want, "\n")), issueTotal); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshRetrySoon(t *testing.T) {
	f := &fakeGitHub{t: t, totals: testTotals(), failures: map[string]int{query("is:open"): 1}}
	c := newTestCollector(t, f)

	// A count that failed without an earlier value is retried soon.
	if !c.refresh(context.Background()) {
		t.Fatal("a count without any value must ask for a quick retry")
	}
	if c.refresh(context.Background()) {
		t.Fatal("a complete refresh must not ask for a quick retry")
	}

	// A failed count that keeps its earlier value waits for the next refresh.
	f.failures = map[string]int{query("is:open"): 1}
	if c.refresh(context.Background()) {
		t.Fatal("a failed count with an earlier value must not ask for a quick retry")
	}
}
