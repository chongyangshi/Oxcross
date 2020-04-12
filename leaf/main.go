package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/monzo/slog"
	"github.com/monzo/terrors"
	"github.com/monzo/typhon"

	"github.com/icydoge/oxcross/types"
)

var (
	defaultTimeout       = 5
	defaultInterval      = 10
	configReloadInterval = 60
	cfg                  = types.Config{}
	cfgMutex             = sync.RWMutex{}
	configAPIBase        = ""
	leafID               = ""
)

func setConfig(c types.Config) {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	cfg = c
}

func readConfig() types.Config {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	return cfg
}

func loadConfig(ctx context.Context) ([]byte, error) {
	configReq := typhon.NewRequest(ctx, http.MethodGet, fmt.Sprintf("%s/config", configAPIBase), nil)
	configRsp := configReq.Send().Response()
	if configRsp.Error != nil {
		slog.Error(ctx, "Oxcross cannot load config, configserver returned %+v", configRsp.Error)
		return nil, configRsp.Error
	}

	configBody, err := configRsp.BodyBytes(true)
	if err != nil {
		slog.Error(ctx, "Oxcross error reading config response, cannot start: %v", err)
		return nil, err
	}

	return configBody, nil
}

func main() {
	ctx := context.Background()

	if os.Getenv("OXCROSS_LEAF_ID") != "" {
		leafID = os.Getenv("OXCROSS_LEAF_ID")
	} else {
		// If we don't have an explicit leaf ID set, retrieve hostname as leaf ID on best effort basis.
		leafID, _ = os.Hostname()
	}

	// Retrieve config from configserver
	if os.Getenv("OXCROSS_CONFIG_API_BASE") != "" {
		configAPIBase = os.Getenv("OXCROSS_CONFIG_API_BASE")
		slog.Info(ctx, "Oxcross loading config from %s", configAPIBase)
	} else {
		err := terrors.InternalService("no_config_api", "Oxcross config API not set in OXCROSS_CONFIG_API_BASE", nil)
		slog.Critical(ctx, "Oxcross cannot start: %+v", err)
		panic(err)
	}

	configBody, err := loadConfig(ctx)
	if err != nil {
		slog.Critical(ctx, "Oxcross cannot start as config load failed: %v", err)
		panic(err)
	}

	c, err := types.ParseConfig(ctx, configBody)
	if err != nil {
		slog.Critical(ctx, "Oxcross cannot start as config parse failed: %v", err)
		panic(err)
	}

	setConfig(*c)

	// Initialize client
	if err = initProbes(ctx); err != nil {
		slog.Critical(ctx, "Oxcross error initializing client: %v, cannot continue", err)
		panic(err)
	}

	// Initialize metrics server
	initMetricsServer()

	// Automatically reload config
	configTicker := time.NewTicker(time.Duration(configReloadInterval) * time.Second)
	go func() {
		for range configTicker.C {
			b, err := loadConfig(ctx)
			if err != nil {
				slog.Error(ctx, "Failed loading up-to-date config: %v, retaining existing config", err)
			}

			c, err := types.ParseConfig(ctx, b)
			if err != nil {
				slog.Error(ctx, "Failed parsing up-to-date config: %v, retaining existing config", err)
			}

			setConfig(*c)
			slog.Debug(ctx, "Reloaded config at %s", time.Now().Format(time.RFC3339), nil)
		}
	}()

	// Log termination gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	slog.Info(ctx, "Oxcross client shutting down")
}
