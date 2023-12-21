package main

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	pb "github.com/primevprotocol/mev-commit/gen/go/rpc/bidderapi/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr = flag.String(
		"server-addr",
		"localhost:13524",
		"The server address in the format of host:port",
	)
	rpcAddr = flag.String(
		"rpc-addr",
		"localhost:13524",
		"The server address in the format of host:port",
	)
	logLevel        = flag.String("log-level", "debug", "Verbosity level (debug|info|warn|error)")
	httpPort        = flag.Int("http-port", 8080, "The port to serve the HTTP metrics endpoint on")
	parallelWorkers = flag.Int("parallel-workers", 5, "The number of parallel workers to run")
)

var (
	receivedPreconfs = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "bidder_emulator",
		Name:      "total_received_preconfs",
		Help:      "Total number of preconfs received from mev_commit nodes",
	})
	sentBids = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "bidder_emulator",
		Name:      "total_sent_bids",
		Help:      "Total number of bids sent to mev_commit nodes",
	})
	sendBidSuccessDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mev_commit",
		Subsystem: "bidder_emulator",
		Name:      "send_bid_success_duration",
		Help:      "Duration of SendBid operation in ms.",
	})
	sendBidFailureDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "mev_commit",
		Subsystem: "bidder_emulator",
		Name:      "send_bid_failure_duration",
		Help:      "Duration of failed SendBid operation in ms.",
	})
)

func main() {
	flag.Parse()
	if *serverAddr == "" {
		fmt.Println("Please provide a valid server address with the -serverAddr flag")
		return
	}

	logger := newLogger(*logLevel)

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		receivedPreconfs,
		sentBids,
		sendBidSuccessDuration,
		sendBidFailureDuration,
	)

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", "err", err)
		}
	}()

	rpcClient, err := ethclient.Dial(*rpcAddr)
	if err != nil {
		logger.Error("failed to connect to rpc", "err", err)
		return
	}

	conn, err := grpc.Dial(
		*serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to connect to server", "err", err)
		return
	}
	defer conn.Close()

	bidderClient := pb.NewBidderClient(conn)

	wg := sync.WaitGroup{}

	for i := 0; i < *parallelWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				err = sendBid(bidderClient, logger, rpcClient)
				if err != nil {
					logger.Error("failed to send bid", "err", err)
				}
				time.Sleep(1 * time.Second)
			}
		}()
	}

	wg.Wait()
}

func sendBid(
	bidderClient pb.BidderClient,
	logger *slog.Logger,
	rpcClient *ethclient.Client,
) error {
	blkNum, err := rpcClient.BlockNumber(context.Background())
	if err != nil {
		logger.Error("failed to get block number", "err", err)
		return err
	}

	randBytes := make([]byte, 32)
	_, err = cryptorand.Read(randBytes)
	if err != nil {
		return err
	}
	amount := rand.Int63n(200000)
	amount += 100000
	// try to keep txn hash unique
	txHash := crypto.Keccak256Hash(
		append(
			randBytes,
			append(
				[]byte(fmt.Sprintf("%d", amount)),
				[]byte(fmt.Sprintf("%d", blkNum))...,
			)...,
		),
	)

	bid := &pb.Bid{
		TxHash:      txHash.Hex(),
		Amount:      amount,
		BlockNumber: int64(blkNum) + 5,
	}

	logger.Info("sending bid", "bid", bid)

	start := time.Now()
	rcv, err := bidderClient.SendBid(context.Background(), bid)
	if err != nil {
		logger.Error("failed to send bid", "err", err)
		return err
	}

	sentBids.Inc()

	preConfCount := 0
	for {
		_, err := rcv.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("failed receiving preconf", "error", err)
			sendBidFailureDuration.Set(float64(time.Since(start).Milliseconds()))
			return err
		}
		receivedPreconfs.Inc()
		preConfCount++
	}

	sendBidSuccessDuration.Set(float64(time.Since(start).Milliseconds()))
	return nil
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
