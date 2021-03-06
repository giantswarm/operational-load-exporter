package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	opsgenieCollector, err := NewOpsgenieCollector(os.Getenv("OPSGENIE_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(opsgenieCollector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8000", nil))
}
