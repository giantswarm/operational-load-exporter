package main

import (
	"context"
	"log"

	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/prometheus/client_golang/prometheus"
)

type AlertGroupType string

type AlertGroup struct {
	Name  string
	Query string
	Team  string
	Type  AlertGroupType
}

type opsgenieCollector struct {
	opsgenieClient *alert.Client

	alertCounter *prometheus.Desc

	alertGroups []AlertGroup
}

var (
	BusinessHours      AlertGroupType = "business_hours"
	OutOfBusinessHours AlertGroupType = "out_of_business_hours"
	AllHours           AlertGroupType = "all_hours"
)

func NewOpsgenieCollector(apiKey string) (*opsgenieCollector, error) {
	opsgenieClient, err := alert.NewClient(&client.Config{ApiKey: apiKey})
	if err != nil {
		return nil, err
	}

	collector := &opsgenieCollector{
		opsgenieClient: opsgenieClient,

		alertCounter: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "opsgenie", "alert_total"),
			"Number of alerts in Opsgenie",
			[]string{"name", "type", "team"},
			nil,
		),

		alertGroups: []AlertGroup{
			{
				Name:  "all",
				Query: "",
				Team:  "",
				Type:  AllHours,
			},
			{
				Name:  "urgent_email",
				Query: "source: urgent-email or source: kaas-urgent-email",
				Team:  "",
				Type:  AllHours,
			},
			{
				Query: "responders: atlas and not responders: empowerment and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "atlas",
				Type:  BusinessHours,
			},
			{
				Query: "responders: cabbage and not responders: empowerment and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "cabbage",
				Type:  BusinessHours,
			},
			{
				Query: "responders: honeybadger and not responders: empowerment and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "honeybadger",
				Type:  BusinessHours,
			},
			{
				Query: "responders: phoenix and not responders: cloud_kaas and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "phoenix",
				Type:  BusinessHours,
			},
			{
				Query: "responders: rainbow and not responders: cloud_kaas and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "rainbow",
				Type:  BusinessHours,
			},
			{
				Query: "responders: rocket and not responders: onprem_kaas and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "rocket",
				Type:  BusinessHours,
			},
			{
				Query: "responders: cloud_kaas and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "cloud_kaas",
				Type:  OutOfBusinessHours,
			},
			{
				Query: "responders: empowerment and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "cloud_native_packs",
				Type:  OutOfBusinessHours,
			},
			{
				Query: "responders: onprem_kaas and not (source: urgent-email or source: kaas-urgent-email)",
				Team:  "onprem_kaas",
				Type:  OutOfBusinessHours,
			},
		},
	}

	return collector, nil
}

func (c *opsgenieCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.alertCounter
}

func (c *opsgenieCollector) Collect(ch chan<- prometheus.Metric) {
	for _, alertGroup := range c.alertGroups {
		result, err := c.opsgenieClient.CountAlerts(context.Background(), &alert.CountAlertsRequest{
			Query: alertGroup.Query,
		})
		if err != nil {
			log.Print(err)
		}

		name := alertGroup.Name
		if name == "" {
			name = alertGroup.Team
		}

		ch <- prometheus.MustNewConstMetric(
			c.alertCounter,
			prometheus.CounterValue,
			float64(result.Count),
			name, string(alertGroup.Type), alertGroup.Team,
		)
	}
}
