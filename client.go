package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	client *http.Client

	dnsLatencyVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "http_client",
			Name:      "dns_duration_seconds",
			Help:      "Trace dns latency histogram.",
			Buckets:   []float64{.005, .01, .025, .05},
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
		trace := &promhttp.InstrumentTrace{
			DNSStart: func(t float64) {
				dnsLatencyVec.WithLabelValues("dns_start").Observe(t)
			},
			DNSDone: func(t float64) {
				dnsLatencyVec.WithLabelValues("dns_done").Observe(t)
			},
			// TLSHandshakeStart: func(t float64) {
			// 	tlsLatencyVec.WithLabelValues("tls_handshake_start").Observe(t)
			// },
			// TLSHandshakeDone: func(t float64) {
			// 	tlsLatencyVec.WithLabelValues("tls_handshake_done").Observe(t)
			// },
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

func fill(p pixel) error {
	u := "http://" + ip + "/fill?colour=" +
		url.QueryEscape(p.Hex())
	resp, err := getClient().Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func postRaw(p []pixel) error {
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}
	u := "http://" + ip + "/raw"
	body := bytes.NewBuffer(j)

	resp, err := getClient().Post(u, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func clear() error {
	u := "http://" + ip + "/clear"
	start := time.Now()
	resp, err := getClient().Get(u)
	end := time.Now()
	delta := end.Sub(start)
	log.Printf("took %v", delta)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// fmt.Printf("%+v\n", resp)
	return nil
}

func fadeOut(p pixel, d time.Duration) error {
	return fade(p, mustParseHex("#000000"), d)
}

func fade(p pixel, toPixel pixel, d time.Duration) error {

	var steps float64
	var t time.Duration
	// use a minimum step time of 200ms
	if d > 600*time.Millisecond {
		t = 200 * time.Millisecond
		steps = float64(d / t)
	} else {
		steps = 4
		t = d / time.Duration(steps)
	}

	fadeStart := time.Now()
	for i := 1.0; i <= steps; i++ {
		end := time.Now().Add(t)
		step := i / steps
		np := pixel{p.BlendHcl(toPixel.Color, step)}
		// log.Printf("%v / %v == %v        t%s", i, steps, step, np.Color.Hex())
		// err := fill(pixel{gamma(np.Color, 0.5)})
		err := fill(np)
		if err != nil {
			return err
		}
		left := time.Until(end)
		// log.Printf("sleeping %v", left)
		time.Sleep(left)
	}
	fadeEnd := time.Now()
	fadeHist.Observe(fadeEnd.Sub(fadeStart).Seconds())
	fadeSumm.Observe(fadeEnd.Sub(fadeStart).Seconds())
	return nil
}
