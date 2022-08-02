package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	incidentCollector, err := NewIncidentCollector(os.Getenv("INCIDENT_IO_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	opsgenieCollector, err := NewOpsgenieCollector(os.Getenv("OPSGENIE_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(incidentCollector)
	prometheus.MustRegister(opsgenieCollector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8000", nil))
}
