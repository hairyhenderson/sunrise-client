package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	client *http.Client

	// inFlightGauge = promauto.NewGauge(prometheus.GaugeOpts{
	// 	Namespace: "http_client",
	// 	Name:      "in_flight_requests",
	// 	Help:      "A gauge of in-flight requests for the wrapped client.",
	// })

	// counter = promauto.NewCounterVec(
	// 	prometheus.CounterOpts{
	// 		Namespace: "http_client",
	// 		Name:      "api_requests_total",
	// 		Help:      "A counter for requests from the wrapped client.",
	// 	},
	// 	[]string{"code", "method"},
	// )

	// dnsLatencyVec uses custom buckets based on expected dns durations.
	// It has an instance label "event", which is set in the
	// DNSStart and DNSDonehook functions defined in the
	// InstrumentTrace struct below.
	dnsLatencyVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "http_client",
			Name:      "dns_duration_seconds",
			Help:      "Trace dns latency histogram.",
			Buckets:   []float64{.005, .01, .025, .05},
		},
		[]string{"event"},
	)

	// tlsLatencyVec uses custom buckets based on expected tls durations.
	// It has an instance label "event", which is set in the
	// TLSHandshakeStart and TLSHandshakeDone hook functions defined in the
	// InstrumentTrace struct below.
	tlsLatencyVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "http_client",
			Name:      "tls_duration_seconds",
			Help:      "Trace tls latency histogram.",
			Buckets:   []float64{.05, .1, .25, .5},
		},
		[]string{"event"},
	)

	histVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "http_client",
			Name:      "request_duration_seconds",
			Help:      "A histogram of request latencies.",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method"},
	)

	summVec = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "http_client",
			Name:       "request_duration_quantile_seconds",
			Help:       "A histogram of request latencies.",
			Objectives: map[float64]float64{0.1: 0.1, 0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 0.999: 0.0001},
		},
		[]string{"code", "method"},
	)
)

func getClient() *http.Client {
	if client == nil {
		// Define functions for the available httptrace.ClientTrace hook
		// functions that we want to instrument.
		trace := &promhttp.InstrumentTrace{
			DNSStart: func(t float64) {
				dnsLatencyVec.WithLabelValues("dns_start").Observe(t)
			},
			DNSDone: func(t float64) {
				dnsLatencyVec.WithLabelValues("dns_done").Observe(t)
			},
			TLSHandshakeStart: func(t float64) {
				tlsLatencyVec.WithLabelValues("tls_handshake_start").Observe(t)
			},
			TLSHandshakeDone: func(t float64) {
				tlsLatencyVec.WithLabelValues("tls_handshake_done").Observe(t)
			},
		}

		// Wrap the default RoundTripper with middleware.
		client = &http.Client{
			Transport:
			// promhttp.InstrumentRoundTripperInFlight(inFlightGauge,
			// 	promhttp.InstrumentRoundTripperCounter(counter,
			promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(summVec,
					promhttp.InstrumentRoundTripperDuration(histVec, http.DefaultTransport),
				),
			),
			// 	),
			// ),
		}
	}

	return client
}
