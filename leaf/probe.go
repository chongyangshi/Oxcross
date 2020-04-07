package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/monzo/terrors"

	"github.com/monzo/slog"
	"github.com/monzo/typhon"
	"golang.org/x/sync/errgroup"

	"github.com/icydoge/oxcross/types"
)

var (
	cache       tokenCache
	probeClient typhon.Service
)

type tokenCache map[string]tokenCacheEntry

type tokenCacheEntry struct {
	ID    string
	Token string
	Time  string
}

func init() {
	cache = tokenCache{}
}

func initProbes(ctx context.Context, cfg *Config) error {
	// Do not reuse connections to get accurate full handshake times
	roundTripper := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
		DialContext: (&net.Dialer{
			Timeout:   time.Second * time.Duration(cfg.Timeout),
			KeepAlive: -1 * time.Second, // Disabled
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   time.Second * time.Duration(cfg.Timeout),
		ResponseHeaderTimeout: time.Second * time.Duration(cfg.Timeout),
		ExpectContinueTimeout: 1 * time.Second,
	}

	probeClient = typhon.HttpService(roundTripper).Filter(typhon.ExpirationFilter).Filter(typhon.H2cFilter).Filter(typhon.ErrorFilter)

	// Main outgoing routine
	outgoingTicker := time.NewTicker(time.Second * time.Duration(cfg.Interval))
	go func() {
		for range outgoingTicker.C {
			g, ctx := errgroup.WithContext(ctx)

			for _, origin := range cfg.Origins {
				origin := origin // Avoids shadowing
				g.Go(func() error {
					start := time.Now()
					r := typhon.NewRequest(ctx, http.MethodGet, origin.URL, nil).Send().Response()
					if r.Error == nil {
						slog.Error(ctx, "Error received from %s %s:%d: %d %v", origin.Scheme, origin.Hostname, origin.Port, r.StatusCode, r.Error)
						return r.Error
					}
					end := time.Now()
					duration := start.Sub(end)

					// Success
					rsp := &types.OriginResponse{}
					rBytes, err := r.BodyBytes(true)
					if err != nil {
						slog.Error(ctx, "Error parsing response from %s %s:%d: %v", origin.Scheme, origin.Hostname, origin.Port, err)
						return err
					}

					err = json.Unmarshal(rBytes, rsp)
					if err != nil {
						slog.Error(ctx, "Error parsing response from %s %s:%d: %v", origin.Scheme, origin.Hostname, origin.Port, err)
						return err
					}

					cacheSearch, found := cache[origin.URL]
					if found && cacheSearch.Token == rsp.Token {
						// If this is not the first time we process this origin, check we've not received any repeated token.
						// If this happens, it will mean a bad cache and not a true server response, whose token should be
						// guaranteed to be unique on each response.
						err = terrors.BadResponse("repeated_token", fmt.Sprintf("Received repeated token from origin %s: %s at %s", rsp.Identifier, rsp.Token, rsp.ServerTime), nil)
						slog.Error(ctx, "%+v", err)
						return err
					}

					cache[origin.URL] = tokenCacheEntry{
						ID:    rsp.Identifier,
						Token: rsp.Token,
						Time:  rsp.ServerTime,
					}

					// Estimate server time drift with 1/2 of response time. This is not scientific but we have no better data.
					serverTime, err := time.Parse(time.RFC3339, rsp.ServerTime)
					if err != nil {
						slog.Error(ctx, "Unexpected error parsing response server time %s from %s %s:%d: %v", rsp.ServerTime, origin.Scheme, origin.Hostname, origin.Port, err)
						return err
					}

					estimatedDrift := serverTime.Sub(start.Add(duration / 2))
					registerOriginTimeDrift(rsp.Identifier, cfg.SourceID, estimatedDrift.Seconds())

					registerProbeTiming(rsp.Identifier, cfg.SourceID, duration.Seconds())

					return nil

				})
			}
			if err := g.Wait(); err != nil {
				slog.Error(ctx, "Error sending probing to at least one origin: %v", err)
			}
		}
	}()
	return nil

}
