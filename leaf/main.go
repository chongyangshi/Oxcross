package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/monzo/typhon"

	"github.com/monzo/slog"
	"github.com/monzo/terrors"

	"github.com/icydoge/oxcross/types"
)

var (
	defaultTimeout  = 5
	defaultInterval = 10
	cfg             = types.Config{}
)

func main() {
	ctx := context.Background()
	configAPIBase := ""

	// Retrieve config from configserver
	if os.Getenv("OXCROSS_CONFIG_API_BASE") != "" {
		configAPIBase = os.Getenv("OXCROSS_CONFIG_API_BASE")
		slog.Info(ctx, "Oxcross loading config from %s", configAPIBase)
	} else {
		err := terrors.InternalService("no_config_api", "Oxcross config API not set in OXCROSS_CONFIG_API_BASE", nil)
		slog.Critical(ctx, "Oxcross cannot start: %+v", err)
		panic(err)
	}

	configReq := typhon.NewRequest(ctx, http.MethodGet, fmt.Sprintf("%s/config", configAPIBase), nil)
	configRsp := configReq.Send().Response()
	if configRsp.Error != nil {
		slog.Critical(ctx, "Oxcross cannot load config, configserver returned %+v", configRsp.Error)
		panic(configRsp.Error)
	}

	configBody, err := configRsp.BodyBytes(true)
	if err != nil {
		slog.Critical(ctx, "Oxcross error reading config response, cannot start: %v", err)
		panic(err)
	}

	types.MustLoadConfig(ctx, configBody)

	// Initialize client
	if err = initProbes(ctx, &cfg); err != nil {
		slog.Critical(ctx, "Oxcross error initializing client: %v, cannot continue", err)
		panic(err)
	}

	// Initialize metrics server
	initMetricsServer()

	// Log termination gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	slog.Info(ctx, "Oxcross client shutting down")
}
