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

	"github.com/chongyangshi/oxcross/types"
)

// Simple typhon server responding to polls from clients
func service() typhon.Service {
	router := typhon.Router{}
	router.GET("/oxcross", serveResponse)
	router.GET("/healthz", serveResponse)

	svc := router.Serve().Filter(typhon.ErrorFilter).Filter(typhon.H2cFilter)

	return svc
}

func serveResponse(req typhon.Request) typhon.Response {
	serverTime := time.Now().Format(time.RFC3339)

	entropy := make([]byte, 24)
	rand.Read(entropy) // Best effort
	saltedHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", serverTime, string(entropy))))
	token := hex.EncodeToString(saltedHash[:])

	rsp := types.OriginResponse{
		ServerTime: serverTime,
		Token:      token,
	}

	return req.Response(&rsp)
}

func main() {
	ctx := context.Background()

	port := types.OriginServerPort
	envPort := os.Getenv("OXCROSS_ORIGIN_PORT")
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
