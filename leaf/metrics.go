package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	recentRestartsWindow = time.Minute * 10
	maxRecentRestarts    = 5
)

var (
	probeTimings = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "oxcross_leaf",
		Name:      "probe_timings",
		Help:      "Record the result of a successful probe to an origin",
		Buckets:   []float64{0, 0.05, 0.1, 0.5, 1, 2},
	}, []string{"origin_id", "source_id"})
)

var (
	originTimeDrifts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "oxcross_leaf",
		Name:      "origin_time_drift",
		Help:      "Record the perceived timedrift of the origin server",
	}, []string{"origin_id", "source_id"})
)

func registerProbeTiming(originID, sourceID string, timing float64) {
	probeTimings.WithLabelValues(originID, sourceID).Observe(timing)
}

func registerOriginTimeDrift(originID, sourceID string, timeDirft float64) {
	originTimeDrifts.WithLabelValues(originID, sourceID).Set(timeDirft)
}

func initMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9299", nil)
}
