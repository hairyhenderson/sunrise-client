package main

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
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

func (c *PixelStripClient) getClient() *http.Client {
	if c.client == nil {
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
		c.client = &http.Client{
			Transport: promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(summVec,
					promhttp.InstrumentRoundTripperDuration(histVec, http.DefaultTransport),
				),
			),
		}
	}

	return c.client
}

// PixelStripClient -
type PixelStripClient struct {
	url    *url.URL
	client *http.Client
}

// Send - send a command without expecting any response
func (c *PixelStripClient) Send(path string, args map[string]string) error {
	rel, err := url.Parse(path)
	if err != nil {
		return err
	}
	u := c.url.ResolveReference(rel)
	q := u.Query()
	for k, v := range args {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	resp, err := c.getClient().Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SendBody - send a command with a body, without expecting any response
func (c *PixelStripClient) SendBody(path string, args map[string]string, body []byte) error {
	buf := bytes.NewBuffer(body)

	rel, err := url.Parse(path)
	if err != nil {
		return err
	}
	u := c.url.ResolveReference(rel)
	q := u.Query()
	for k, v := range args {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := c.getClient().Post(u.String(), "application/json", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
