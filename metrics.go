package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ns       = "sunrise_client"
	fadeHist = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "fade_duration_seconds",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
	})
	fadeSumm = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:  ns,
		Name:       "fade_duration_quantile_seconds",
		Objectives: map[float64]float64{0.1: 0.1, 0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 0.999: 0.0001},
	})
)

func listenMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
