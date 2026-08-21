package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// TEAM_METRIC_LABEL is the Prometheus label every collector uses to break
	// its metric down per team. Not to be confused with TEAM_LABEL_PREFIX,
	// which is a GitHub issue label prefix.
	TEAM_METRIC_LABEL = "team"
)

func main() {
	githubCollector, err := NewGitHubCollector(os.Getenv("GITHUB_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(githubCollector)

	incidentCollector, err := NewIncidentCollector(os.Getenv("INCIDENT_IO_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(incidentCollector)

	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              ":8000",
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
