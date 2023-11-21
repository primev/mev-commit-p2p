package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"

	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	serverAddr = flag.String(
		"server-addr",
		"localhost:13524",
		"The server address in the format of host:port",
	)
	logLevel        = flag.String("log-level", "debug", "Verbosity level (debug|info|warn|error)")
	httpPort        = flag.Int("http-port", 8080, "The port to serve the HTTP metrics endpoint on")
	errorProbablity = flag.Int(
		"error-probability", 0, "The probability of returning an error when sending a bid response",
	)
)

var (
	receivedBids = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "provider_emulator",
		Name:      "total_received_bids",
		Help:      "Total number of bids received from mev_commit nodes",
	})
	sentBids = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "provider_emulator",
		Name:      "total_sent_bids",
		Help:      "Total number of bids sent mev_commit nodes",
	})
	rejectedBids = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "provider_emulator",
		Name:      "total_rejected_bids",
		Help:      "Total number of bids rejected",
	})
)

func main() {
	flag.Parse()
	if *serverAddr == "" {
		fmt.Println("Please provide a valid server address with the -serverAddr flag")
		return
	}

	logger := newLogger(*logLevel)

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: router,
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(receivedBids, sentBids)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", "err", err)
		}
	}()

	providerClient, err := NewProviderClient(*serverAddr, logger)
	if err != nil {
		logger.Error("failed to create provider client", "error", err)
		return
	}
	defer providerClient.Close()

	err = providerClient.CheckAndStake()
	if err != nil {
		logger.Error("failed to check and stake", "error", err)
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
			receivedBids.Inc()
			logger.Info("received new bid", "bid", bidString(bid))

			status := providerapiv1.BidResponse_STATUS_ACCEPTED
			if *errorProbablity > 0 {
				if rand.Intn(100) < *errorProbablity {
					logger.Warn("sending error response")
					status = providerapiv1.BidResponse_STATUS_REJECTED
					rejectedBids.Inc()
				}
			}
			err := providerClient.SendBidResponse(context.Background(), &providerapiv1.BidResponse{
				BidDigest: bid.BidDigest,
				Status:    status,
			})
			if err != nil {
				logger.Error("failed to send bid response", "error", err)
				return
			}
			sentBids.Inc()
			logger.Info("sent bid", "status", status.String())
		}
	}
}

func bidString(bid *providerapiv1.Bid) string {
	return fmt.Sprintf(
		"bid: {txnHash: %s, block_number: %d, bid_amount: %d, bid_hash: %x}",
		bid.TxHash, bid.BlockNumber, bid.BidAmount, bid.BidDigest,
	)
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
