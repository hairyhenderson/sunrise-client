package main

import (
	"bytes"
	"log"
	"net"
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

	wait := 100 * time.Millisecond
	for i, c := range colors {
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
	}

	printMetrics()

	time.Sleep(2 * time.Second)
	last := colors[len(colors)-1]
	err = fadeOut(last, 5*time.Second)
	if err != nil {
		panic(err)
	}
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
