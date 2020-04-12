package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/monzo/slog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/icydoge/oxcross/types"
)

const (
	recentRestartsWindow = time.Minute * 10
	maxRecentRestarts    = 5
)

var (
	lastRestartTime time.Time
	recentRestarts  int
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

	// A simple automatic recovery routine for the metrics server with limited recent retries
	ctx := context.Background()
	go func() {
		for {
			err := http.ListenAndServe(fmt.Sprintf(":%d", types.ProbeMetricsServerPort), nil)
			if err != nil {
				slog.Error(ctx, "Local metrics server encountered error: %v", err)

				timeOfError := time.Now()

				if timeOfError.Sub(lastRestartTime) > recentRestartsWindow {
					recentRestarts = 0
				}

				if recentRestarts > maxRecentRestarts {
					slog.Critical(ctx, "Too many recent restarts (%d), exiting.", maxRecentRestarts)
					break
				}

				slog.Warn(ctx, "Restaring metrics server following recent error %v", err)
				recentRestarts++
			}
		}
	}()
}
