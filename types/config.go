package types

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/monzo/slog"
	"github.com/monzo/terrors"
)

// In simple mode, we only check that we've received a 200 response,
// which is useful for applying to existing healthz endpoints on
// servers without deploying an Oxcross origin server.
// In advanced mode, we also detect whether returned content could
// have been incorrectly cached enroute, and estimate a clock drift
// between each leaf and each origin.
const (
	OriginModeSimple   = "simple"
	OriginModeAdvanced = "advanced"
)

const (
	defaultTimeout  = 10
	defaultInterval = 10
)

type Config struct {
	Origins  []OriginEntry `json:"origins"`
	Timeout  int           `json:"timeout"`
	Interval int           `json:"interval"`
}

type OriginEntry struct {
	Scheme   string `json:"scheme"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Mode     string `json:"mode"`
	URL      string // To be composed from schme, hostname, and port
}

func ParseConfig(ctx context.Context, configBody []byte) (*Config, error) {
	cfg := Config{}
	err := json.Unmarshal(configBody, &cfg)
	if err != nil {
		slog.Error(ctx, "Oxcross error parsing config %v: %v", configBody, err)
		return nil, err
	}

	switch {
	case cfg.Timeout == 0:
		cfg.Timeout = defaultTimeout
	case cfg.Interval == 0:
		cfg.Interval = defaultInterval
	}
	slog.Info(ctx, "Oxcross loaded %d origins, with timeout %ds, and interval %ds", len(cfg.Origins), cfg.Timeout, cfg.Interval)

	errParams := map[string]string{
		"timeout":  strconv.FormatInt(int64(cfg.Timeout), 10),
		"interval": strconv.FormatInt(int64(cfg.Interval), 10),
	}

	origins := []OriginEntry{}
	for _, origin := range cfg.Origins {
		if origin.Scheme != "http" && origin.Scheme != "https" {
			slog.Warn(ctx, "Oxcross found invalid scheme %s for hostname %s and port %d, skipping", origin.Scheme, origin.Hostname, origin.Port, errParams)
			continue
		}

		if origin.Port < 0 || origin.Port > 32767 {
			slog.Warn(ctx, "Oxcross found invalid port %d for hostname %s and scheme %s, skipping", origin.Port, origin.Hostname, origin.Scheme, errParams)
			continue
		}

		// Default to advanced mode if not set
		mode := origin.Mode
		if origin.Mode == "" {
			mode = OriginModeAdvanced
		}

		// In advanced mode, we retrieve synchronization information from the fixed endpoint
		fullURL := fmt.Sprintf("%s://%s:%d", origin.Scheme, origin.Hostname, origin.Port)
		if mode == OriginModeAdvanced {
			fullURL = fmt.Sprintf("%s/oxcross", fullURL)
		}

		if _, err := url.Parse(fullURL); err != nil {
			slog.Warn(ctx, "Invalid URL parsed: %s, skipping", fullURL)
			continue
		}

		o := origin
		o.URL = fullURL
		o.Mode = mode

		origins = append(origins, o)
	}
	cfg.Origins = origins

	slog.Info(ctx, "Oxcross loaded %d valid origins", len(cfg.Origins))

	if len(cfg.Origins) == 0 {
		err = terrors.InternalService("empty_config", fmt.Sprintf("Oxcross read empty config %v (or entirely invalid), cannot start", cfg), nil)
		slog.Error(ctx, "%v", err)
		return nil, err
	}

	return &cfg, nil
}
