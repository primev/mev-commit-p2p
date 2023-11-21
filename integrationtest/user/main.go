package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	pb "github.com/primevprotocol/mev-commit/gen/go/rpc/userapi/v1"
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
	logLevel = flag.String("log-level", "debug", "Verbosity level (debug|info|warn|error)")
	httpPort = flag.Int("http-port", 8080, "The port to serve the HTTP metrics endpoint on")
)

var (
	receivedPreconfs = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "user_emulator",
		Name:      "total_received_preconfs",
		Help:      "Total number of preconfs received from mev_commit nodes",
	})
	sentBids = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "mev_commit",
		Subsystem: "user_emulator",
		Name:      "total_sent_bids",
		Help:      "Total number of bids sent to mev_commit nodes",
	})
	sendBidDuration = *prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mev_commit",
			Subsystem: "user_emulator",
			Name:      "send_bid_duration",
			Help:      "Duration of method calls.",
		},
		[]string{"status", "no_of_preconfs"},
	)
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
	registry.MustRegister(receivedPreconfs, sentBids)

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

	userClient := pb.NewUserClient(conn)
	err = checkOrStake(userClient, logger)
	if err != nil {
		logger.Error("failed to check or stake", "err", err)
		return
	}

	for {
		err = sendBid(userClient, logger, rpcClient)
		if err != nil {
			logger.Error("failed to send bid", "err", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func checkOrStake(
	userClient pb.UserClient,
	logger *slog.Logger,
) error {
	stakeAmt, err := userClient.GetStake(context.Background(), &pb.EmptyMessage{})
	if err != nil {
		logger.Error("failed to get stake amount", "err", err)
		return err
	}

	logger.Info("stake amount", "stake", stakeAmt.Amount)

	stakedAmt, set := big.NewInt(0).SetString(stakeAmt.Amount, 10)
	if !set {
		logger.Error("failed to parse stake amount")
		return errors.New("failed to parse stake amount")
	}

	if stakedAmt.Cmp(big.NewInt(0)) > 0 {
		logger.Error("user already staked")
		return nil
	}

	_, err = userClient.RegisterStake(context.Background(), &pb.StakeRequest{
		Amount: "10000000000000000000",
	})
	if err != nil {
		logger.Error("failed to register stake", "err", err)
		return err
	}

	logger.Info("staked 10 ETH")

	return nil
}

func sendBid(
	userClient pb.UserClient,
	logger *slog.Logger,
	rpcClient *ethclient.Client,
) error {
	blkNum, err := rpcClient.BlockNumber(context.Background())
	if err != nil {
		logger.Error("failed to get block number", "err", err)
		return err
	}

	randBytes := make([]byte, 32)
	_, err = rand.Read(randBytes)
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
	rcv, err := userClient.SendBid(context.Background(), bid)
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
			sendBidDuration.WithLabelValues(
				"error",
				fmt.Sprintf("%d", preConfCount),
			).Observe(time.Since(start).Seconds())
			return err
		}
		receivedPreconfs.Inc()
		preConfCount++
	}

	sendBidDuration.WithLabelValues(
		"success",
		fmt.Sprintf("%d", preConfCount),
	).Observe(time.Since(start).Seconds())
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
