package main

import (
	"errors"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/prometheus/client_golang/prometheus"
)

type pagerdutyCollector struct {
	pagerdutyClient *pagerduty.Client

	alertCounter *prometheus.Desc
}

func NewPagerdutyCollector(pagerdutyAuthToken string) (*pagerdutyCollector, error) {
	if pagerdutyAuthToken == "" {
		return nil, errors.New("PagerDuty auth token cannot be blank")
	}

	pagerdutyClient := pagerduty.NewClient(pagerdutyAuthToken)

	collector := &pagerdutyCollector{
		pagerdutyClient: pagerdutyClient,

		alertCounter: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "pagerduty", "alert_total"),
			"Number of alerts in PagerDuty",
			[]string{"name", "type", "team"},
			nil,
		),
	}

	return collector, nil
}

func (c *pagerdutyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.alertCounter
}

func (c *pagerdutyCollector) Collect(ch chan<- prometheus.Metric) {}
