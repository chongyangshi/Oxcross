package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/monzo/slog"
	"github.com/monzo/typhon"

	"github.com/icydoge/oxcross/types"
)

// Warning, this server does not currently handle timeouts!
func service() typhon.Service {
	router := typhon.Router{}
	router.GET("/oxcross", serveResponse)
	router.GET("/healthz", serveResponse)

	svc := router.Serve().Filter(typhon.ErrorFilter).Filter(typhon.H2cFilter)

	return svc
}

func serveResponse(req typhon.Request) typhon.Response {
	origin := ""
	if os.Getenv("ORIGIN_ID") != "" {
		origin = os.Getenv("ORIGIN_ID")
	} else {
		// If we don't have an explicit origin ID set, retrieve hostname as origin ID on best effort basis.
		origin, _ = os.Hostname()
	}

	serverTime := time.Now().Format(time.RFC3339)

	entropy := make([]byte, 24)
	rand.Read(entropy) // Best effort
	saltedHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", serverTime, string(entropy))))
	token := hex.EncodeToString(saltedHash[:])

	rsp := types.OriginResponse{
		Identifier: origin,
		ServerTime: serverTime,
		Token:      token,
	}

	return req.Response(&rsp)
}

func main() {
	ctx := context.Background()

	port := 9301
	envPort := os.Getenv("ORIGIN_PORT")
	if envPort != "" {
		portNum, err := strconv.ParseInt(envPort, 10, 64)
		if err != nil || portNum < 1 || portNum > 32767 {
			slog.Critical(ctx, "Invalid port: %s, cannot initialize", envPort)
			panic(err)
		}
		port = int(portNum)
	}

	// Initialise server for incoming requests
	svc := service()
	srv, err := typhon.Listen(svc, fmt.Sprintf(":%d", port))
	if err != nil {
		slog.Critical(ctx, "Error initializing listener: %v", err)
		panic(err)
	}

	slog.Info(ctx, "Origin server listening on %v", srv.Listener().Addr())

	// Log termination gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
	slog.Info(ctx, "Origin server shutting down")
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Stop(c)
}
