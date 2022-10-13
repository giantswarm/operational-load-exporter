package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
)

type IssueGroupType string

type IssueGroup struct {
	Query string
	Team  string
	Type  IssueGroupType
}

type githubCollector struct {
	githubClient *github.Client

	issueCounter *prometheus.Desc

	issueGroups []IssueGroup
}

const (
	REPO_ID int64 = 30184375 // giantswarm/giantswarm

	TEAM_LABEL_PREFIX = "team/"

	BASE_QUERY = "is:issue repo:giantswarm/giantswarm"
)

var (
	TEAM_LABEL_BLOCKLIST = []string{
		"null",
		"team",
		"teddyfriends",
	}

	All         IssueGroupType = "all"
	Postmortems IssueGroupType = "postmortem"
)

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}

	return false
}

func NewGitHubCollector(accessToken string) (*githubCollector, error) {
	if accessToken == "" {
		return nil, errors.New("github access token cannot be blank")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

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

	labelsSearchResult, _, err := client.Search.Labels(ctx, REPO_ID, TEAM_LABEL_PREFIX, &github.SearchOptions{})
	if err != nil {
		return nil, err
	}
	for _, label := range labelsSearchResult.Labels {
		fullLabelName := label.GetName()
		labelName := strings.TrimPrefix(label.GetName(), TEAM_LABEL_PREFIX)

		if !strings.HasPrefix(fullLabelName, TEAM_LABEL_PREFIX) {
			continue
		}
		if contains(TEAM_LABEL_BLOCKLIST, labelName) {
			continue
		}

		issueGroups = append(issueGroups, IssueGroup{
			Query: "label:postmortem " + "label:" + label.GetName(),
			Team:  labelName,
			Type:  Postmortems,
		})
	}

	collector := &githubCollector{
		githubClient: client,

		issueCounter: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "github", "issue_total"),
			"Number of issues in GitHub",
			[]string{"type", "team", "state"},
			nil,
		),

		issueGroups: issueGroups,
	}

	return collector, nil
}

func (c *githubCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.issueCounter
}

func (c *githubCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	for _, issueGroup := range c.issueGroups {
		for _, state := range []string{"open", "closed"} {
			parts := []string{BASE_QUERY, "is:" + state}

			if issueGroup.Query != "" {
				parts = append(parts, issueGroup.Query)
			}
			query := strings.Join(parts, " ")

			result, _, err := c.githubClient.Search.Issues(ctx, query, &github.SearchOptions{
				ListOptions: github.ListOptions{Page: 1, PerPage: 1}},
			)
			if err != nil {
				log.Print(err)
				break
			}

			ch <- prometheus.MustNewConstMetric(
				c.issueCounter,
				prometheus.GaugeValue,
				float64(*result.Total),
				string(issueGroup.Type), issueGroup.Team, state,
			)
		}
	}
}
