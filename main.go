package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ip = "10.0.1.107"
	ns = "sunrise_client"

	tickHist = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "tick_duration_seconds",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
	})
	tickSumm = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace:  ns,
		Name:       "tick_duration_quantile_seconds",
		Objectives: map[float64]float64{0.1: 0.1, 0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 0.999: 0.0001},
	})

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

func main() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		log.Printf("start")
		ips, err := net.LookupIP("neopixel.local")
		if err != nil {
			panic(err)
		}
		ip = ips[0].String()
		log.Printf("resolved neopixel.local to %s", ip)
	}()
	// log.Printf("neopixels are at %s", ip)

	// u := fmt.Sprintf("http://%s/raw", ip)
	// resp, err := http.Get(u)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Printf("%v", resp)

	wait := 100 * time.Millisecond
	for i, c := range colors {
		// p := make([]pixel, 30)
		// for i := 0; i < 30; i++ {
		// 	p[i] = c
		// }
		// postRaw(p)
		// log.Printf("Step %d", i)
		start := time.Now()
		var from pixel
		if i == 0 {
			from = c
		} else {
			from = colors[i-1]
		}
		err := fade(from, c, wait)
		if err != nil {
			panic(err)
		}
		end := time.Now()
		observeTick(end.Sub(start))
		// log.Printf("took %v", end.Sub(start))
	}

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		panic(err)
	}
	out := &bytes.Buffer{}
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(out, mf); err != nil {
			panic(err)
		}
	}
	log.Printf("Metrics\n%v", out.String())

	time.Sleep(2 * time.Second)
	last := colors[len(colors)-1]
	err = fadeOut(last, 5*time.Second)
	if err != nil {
		panic(err)
	}
}

func observeTick(d time.Duration) {
	tickHist.Observe(d.Seconds())
	tickSumm.Observe(d.Seconds())
}

func fill(p pixel) error {
	u := "http://" + ip + "/fill?colour=" +
		url.QueryEscape(p.Hex())
	// start := time.Now()
	resp, err := getClient().Get(u)
	// end := time.Now()
	// delta := end.Sub(start)
	// log.Printf("took %v", delta)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// fmt.Printf("%+v\n", resp)
	return nil
}

func postRaw(p []pixel) error {
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}
	// fmt.Printf("json: %s\n", string(j))
	u := "http://" + ip + "/raw"
	body := bytes.NewBuffer(j)

	resp, err := getClient().Post(u, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// fmt.Printf("%+v\n", resp)
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
