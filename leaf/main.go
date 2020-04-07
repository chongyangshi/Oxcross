package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/monzo/slog"
	"github.com/monzo/terrors"
)

var (
	configPath = "/etc/oxcross/leaf.conf"
	timeout    = 5
	interval   = 10
	cfg        = Config{}
)

type Config struct {
	SourceID string         `json:"source_id"`
	Origins  []*OriginEntry `json:"origins"`
	Timeout  int            `json:"timeout"`
	Interval int            `json:"interval"`
}

type OriginEntry struct {
	Scheme   string `json:"scheme"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	URL      string
}

func main() {
	ctx := context.Background()

	if os.Getenv("OXCROSS_CONF") != "" {
		configPath = os.Getenv("OXCROSS_CONF")
	}
	slog.Info(ctx, "Oxcross using config from %s", configPath)

	originConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		slog.Critical(ctx, "Oxcross error reading config %s, cannot start: %v", configPath, err)
		panic(err)
	}

	err = json.Unmarshal(originConfig, &cfg)
	if err != nil {
		slog.Critical(ctx, "Oxcross error parsing config %v, cannot start: %v", originConfig, err)
		panic(err)
	}

	switch {
	case cfg.Timeout == 0:
		cfg.Timeout = timeout
	case cfg.Interval == 0:
		cfg.Interval = interval
	}
	slog.Info(ctx, "Oxcross loaded %d origins, with timeout %ds, and interval %ds", len(cfg.Origins), cfg.Timeout, cfg.Interval)

	errParams := map[string]string{
		"timeout":     strconv.FormatInt(int64(cfg.Timeout), 10),
		"interval":    strconv.FormatInt(int64(cfg.Interval), 10),
		"config_path": configPath,
	}

	origins := []*OriginEntry{}
	for _, origin := range cfg.Origins {
		if origin.Scheme != "http" && origin.Scheme != "https" {
			slog.Warn(ctx, "Oxcross found invalid scheme %s for hostname %s and port %d, skipping", origin.Scheme, origin.Hostname, origin.Port, errParams)
			continue
		}

		if origin.Port < 0 || origin.Port > 32767 {
			slog.Warn(ctx, "Oxcross found invalid port %d for hostname %s and scheme %s, skipping", origin.Port, origin.Hostname, origin.Scheme, errParams)
			continue
		}

		fullURL := fmt.Sprintf("%s://%s:%d/oxcross", origin.Scheme, origin.Hostname, origin.Port)
		if _, err := url.Parse(fullURL); err != nil {
			slog.Warn(ctx, "Invalid URL parsed: %s, skipping", fullURL)
			continue
		}

		o := *origin
		o.URL = fullURL
		origins = append(origins, &o)
	}
	cfg.Origins = origins

	slog.Info(ctx, "Oxcross loaded %d valid origins", len(cfg.Origins))

	if len(cfg.Origins) == 0 {
		err = terrors.InternalService("empty_config", fmt.Sprintf("Oxcross read empty config %v (or entirely invalid), cannot start", cfg), nil)
		slog.Critical(ctx, "%v", err)
		panic(err)
	}

	// Initialize client
	if err = initProbes(ctx, &cfg); err != nil {
		slog.Critical(ctx, "Oxcross error initializing client: %v, cannot continue", err, errParams)
		panic(err)
	}

	// Log termination gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	slog.Info(ctx, "Oxcross client shutting down")
}
