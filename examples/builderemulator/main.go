package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/primevprotocol/mev-commit/examples/builderemulator/client"
	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
)

var (
	serverAddr = flag.String(
		"serverAddr",
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

	builderClient, err := client.NewBuilderClient(*serverAddr, logger)
	if err != nil {
		logger.Error("failed to create builder client", "error", err)
		return
	}

	bidS, err := builderClient.ReceiveBids()
	if err != nil {
		logger.Error("failed to create bid receiver", "error", err)
		return
	}

	for {
		select {
		case bid, more := <-bidS:
			if !more {
				logger.Warn("closed bid stream")
				return
			}
			logger.Info("received new bid", "bid", bid)
			err := builderClient.SendBidResponse(context.Background(), &builderapiv1.BidResponse{
				BidHash: bid.BidHash,
				Status:  builderapiv1.BidResponse_STATUS_ACCEPTED,
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
