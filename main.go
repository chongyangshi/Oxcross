package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/monzo/slog"
	"github.com/monzo/typhon"

	"github.com/icydoge/oxcross/types"
)

var cfg = &types.Config{}

// This server runs in Kubernetes and is responsible for distributing origin configurations to leaves.
func service() typhon.Service {
	router := typhon.Router{}
	router.GET("/config", serveConfigResponse)
	router.GET("/healthz", serveLivesss)

	svc := router.Serve().Filter(typhon.ErrorFilter).Filter(typhon.H2cFilter)

	return svc
}

func serveLivesss(req typhon.Request) typhon.Response {
	return req.Response(nil)
}

func serveConfigResponse(req typhon.Request) typhon.Response {
	return req.Response(&cfg)
}

func main() {
	ctx := context.Background()

	configPath := os.Getenv("OXCROSS_CONF")
	slog.Info(ctx, "Oxcross using config from %s", configPath)

	originConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		slog.Critical(ctx, "Error reading config %s, cannot start: %v", configPath, err)
		panic(err)
	}

	cfg, err = types.ParseConfig(ctx, originConfig)
	if err != nil {
		slog.Critical(ctx, "Error parsing config %s, cannot start: %v", configPath, err)
		panic(err)
	}

	// Initialise server for incoming requests
	svc := service()
	srv, err := typhon.Listen(svc, fmt.Sprintf(":%d", types.ConfigServerPort))
	if err != nil {
		slog.Critical(ctx, "Error initializing listener: %v", err)
		panic(err)
	}

	// Log termination gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	slog.Info(ctx, "Origin server shutting down")
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Stop(c)
}
