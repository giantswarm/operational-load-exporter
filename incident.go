package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ListIncidentsResponse struct {
	PaginationMeta PaginationMeta `json:"pagination_meta"`
}

type PaginationMeta struct {
	TotalRecordCount int `json:"total_record_count"`
}

type incidentCollector struct {
	apiKey string

	incidentCounter *prometheus.Desc
}

func NewIncidentCollector(apiKey string) (*incidentCollector, error) {
	if apiKey == "" {
		return nil, errors.New("incident.io api key cannot be blank")
	}

	collector := &incidentCollector{
		apiKey: apiKey,

		incidentCounter: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "incident_io", "incident_total"),
			"Number of incidents in Incident.io",
			[]string{},
			nil,
		),
	}

	return collector, nil
}

func (c *incidentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.incidentCounter
}

func (c *incidentCollector) Collect(ch chan<- prometheus.Metric) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.incident.io/v1/incidents", nil)
	if err != nil {
		log.Print(err)
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Printf("received http code: %v, error: %v", resp.StatusCode, string([]byte(b)))
	}

	listIncidentsResponse := &ListIncidentsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(listIncidentsResponse); err != nil {
		log.Print(err)
	}

	ch <- prometheus.MustNewConstMetric(
		c.incidentCounter,
		prometheus.CounterValue,
		float64(listIncidentsResponse.PaginationMeta.TotalRecordCount),
	)
}
