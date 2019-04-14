package main

import (
	"bytes"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/common/expfmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ip = "neopixel.local"
)

func main() {
	go listenMetrics()

	log.Printf("Looking up IP...")
	ips, err := net.LookupIP(ip)
	if err != nil {
		panic(err)
	}
	ip = ips[0].String()
	log.Printf("resolved to %s", ip)

	strip := NewPixelStrip(&url.URL{
		Scheme: "http",
		Host:   ip,
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGTERM)

	go func() {
		s := <-sigs
		err := strip.Clear()
		if err != nil {
			panic(err)
		}
		switch s := s.(type) {
		case syscall.Signal:
			os.Exit(128 + int(s))
		default:
			os.Exit(1)
		}
	}()

	wait := 10 * time.Millisecond
	for i, c := range colors {
		var from pixel
		if i == 0 {
			from = c
		} else {
			from = colors[i-1]
		}
		err := strip.Fade(from, c, wait)
		if err != nil {
			panic(err)
		}
	}

	time.Sleep(2 * time.Second)
	last := colors[len(colors)-1]
	err = strip.FadeOut(last, 5*time.Second)
	if err != nil {
		panic(err)
	}

	printMetrics()
}

func printMetrics() {
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
}
