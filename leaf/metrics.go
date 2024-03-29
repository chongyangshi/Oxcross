package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/monzo/slog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/chongyangshi/oxcross/types"
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
		Help:      "Record the timing of a successful probe to an origin",
		Buckets:   []float64{0, 0.05, 0.1, 0.5, 1, 2},
	}, []string{"origin_id", "source_id"})
	probeResults = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "oxcross_leaf",
		Name:      "probe_results",
		Help:      "Record the result of an attempted probe to an origin",
	}, []string{"origin_id", "source_id", "result", "reason"})
	originTimeDrifts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "oxcross_leaf",
		Name:      "origin_time_drift",
		Help:      "Record the perceived timedrift of the origin server",
	}, []string{"origin_id", "source_id"})
	originStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "oxcross_leaf",
		Name:      "origin_status",
		Help:      "Record the current status of an origin from the perspective of the probe",
	}, []string{"origin_id", "source_id", "result", "reason"})
)

func registerProbeTiming(originID, sourceID string, timing float64) {
	probeTimings.WithLabelValues(originID, sourceID).Observe(timing)
}

func registerProbeResult(originID, sourceID string, result bool, reason string) {
	probeResults.WithLabelValues(originID, sourceID, strconv.FormatBool(result), reason).Add(1)

	// Also update origin status as a real time value
	gaugeValue := 1.0
	if !result {
		gaugeValue = 0.0
	}
	originStatus.WithLabelValues(originID, sourceID, strconv.FormatBool(result), reason).Set(gaugeValue)
}

func registerOriginTimeDrift(originID, sourceID string, timeDirft float64) {
	originTimeDrifts.WithLabelValues(originID, sourceID).Set(timeDirft)
}

func initMetricsServer() {
	ctx := context.Background()
	http.Handle("/metrics", promhttp.Handler())

	port := types.ProbeMetricsServerPort
	envPort := os.Getenv("OXCROSS_METRICS_PORT")
	if envPort != "" {
		portNum, err := strconv.ParseInt(envPort, 10, 64)
		if err != nil || portNum < 1 || portNum > 32767 {
			slog.Critical(ctx, "Invalid port: %s, cannot initialize", envPort)
			panic(err)
		}
		port = int(portNum)
	}

	// A simple automatic recovery routine for the metrics server with limited recent retries
	go func() {
		for {
			err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
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
