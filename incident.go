package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	TEAM_CUSTOM_FIELD_ID = "01G49Y5JP7RM5Q3CR02G2EQE1Q"
)

type ShowCustomFieldResponse struct {
	CustomField CustomField `json:"custom_field"`
}

type CustomField struct {
	Options []Option `json:"options"`
}

type Option struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type ListIncidentsResponse struct {
	PaginationMeta PaginationMeta `json:"pagination_meta"`
}

type PaginationMeta struct {
	TotalRecordCount int `json:"total_record_count"`
}

type TeamID struct {
	ID   string
	Name string
}

type incidentCollector struct {
	apiKey string

	incidentCounter *prometheus.Desc

	teams []TeamID
}

func NewIncidentCollector(apiKey string) (*incidentCollector, error) {
	if apiKey == "" {
		return nil, errors.New("incident.io api key cannot be blank")
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.incident.io/v1/custom_fields/"+TEAM_CUSTOM_FIELD_ID, nil)
	if err != nil {
		log.Print(err)
	}
	req.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Print(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("received http code: %v, error: %v", resp.StatusCode, string([]byte(b)))
	}

	showCustomFieldResponse := &ShowCustomFieldResponse{}
	if err := json.NewDecoder(resp.Body).Decode(showCustomFieldResponse); err != nil {
		log.Print(err)
	}

	teams := []TeamID{
		TeamID{}, // Empty team to get all incidents.
	}
	for _, option := range showCustomFieldResponse.CustomField.Options {
		teams = append(teams, TeamID{
			ID:   option.ID,
			Name: option.Value,
		})
	}

	collector := &incidentCollector{
		apiKey: apiKey,

		incidentCounter: prometheus.NewDesc(
			prometheus.BuildFQName("operations", "incident_io", "incident_total"),
			"Number of incidents in incident.io",
			[]string{"team"},
			nil,
		),

		teams: teams,
	}

	return collector, nil
}

func (c *incidentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.incidentCounter
}

func (c *incidentCollector) Collect(ch chan<- prometheus.Metric) {
	for _, team := range c.teams {
		client := &http.Client{
			Timeout: time.Second * 5,
		}

		req, err := http.NewRequest(http.MethodGet, "https://api.incident.io/v2/incidents", nil)
		if err != nil {
			log.Print(err)
		}
		req.Header.Add("Authorization", "Bearer "+c.apiKey)

		values := req.URL.Query()
		values.Add("page_size", "1")
		if team.Name != "" {
			values.Add(fmt.Sprintf("custom_field[%v][one_of]", TEAM_CUSTOM_FIELD_ID), team.ID)
		}
		req.URL.RawQuery = values.Encode()

		resp, err := client.Do(req)
		if err != nil {
			log.Print(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
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
			team.Name,
		)
	}
}
