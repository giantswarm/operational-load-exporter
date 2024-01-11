package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	opsgenieCollector, err := NewOpsgenieCollector(os.Getenv("OPSGENIE_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(opsgenieCollector)

	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              ":8000",
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
