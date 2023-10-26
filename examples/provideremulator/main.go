package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/primevprotocol/mev-commit/examples/provideremulator/client"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
)

var (
	serverAddr = flag.String(
		"server-addr",
		"localhost:13524",
		"The server address in the format of host:port",
	)

	logLevel = flag.String("log-level", "debug", "Verbosity level (debug|info|warn|error)")
)

func main() {
	flag.Parse()
	if *serverAddr == "" {
		fmt.Println("Please provide a valid server address with the -serverAddr flag")
		return
	}

	logger := newLogger(*logLevel)

	providerClient, err := client.NewProviderClient(*serverAddr, logger)
	if err != nil {
		logger.Error("failed to create provider client", "error", err)
		return
	}

	bidS, err := providerClient.ReceiveBids()
	if err != nil {
		logger.Error("failed to create bid receiver", "error", err)
		return
	}

	fmt.Printf("connected to provider %s, receiving bids...\n", *serverAddr)

	for {
		select {
		case bid, more := <-bidS:
			if !more {
				logger.Warn("closed bid stream")
				return
			}
			logger.Info("received new bid", "bid", bid)
			err := providerClient.SendBidResponse(context.Background(), &providerapiv1.BidResponse{
				BidDigest: bid.BidDigest,
				Status:    providerapiv1.BidResponse_STATUS_ACCEPTED,
			})
			if err != nil {
				logger.Error("failed to send bid response", "error", err)
				return
			}
			logger.Info("accepted bid")
		}
	}
}

func newLogger(lvl string) *slog.Logger {
	var level = new(slog.LevelVar) // debug by default

	switch lvl {
	case "debug":
		level.Set(slog.LevelDebug)
	case "info":
		level.Set(slog.LevelInfo)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelDebug)
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}