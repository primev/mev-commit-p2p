package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	pb "github.com/primevprotocol/mev-commit/gen/go/rpc/bidderapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	logLevel = flag.String(
		"log-level",
		"debug",
		"Verbosity level (debug|info|warn|error)",
	)
	logFmt = flag.String(
		"log-fmt",
		"text",
		"Format of the log output: 'text', 'json'",
	)
	logTags = flag.String(
		"log-tags",
		"",
		"Comma-separated list of <name:value> pairs that will be inserted into each log line",
	)
	httpPort = flag.Int(
		"http-port",
		8080,
		"The port to serve the HTTP metrics endpoint on",
	)
	parallelWorkers = flag.Int(
		"parallel-workers",
		7,
		"The number of parallel workers to run",
	)
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

	logger, err := util.NewLogger(*logLevel, *logFmt, *logTags, os.Stdout)
	if err != nil {
		fmt.Printf("failed to create logger: %v", err)
		return
	}

	if *serverAddr == "" {
		fmt.Println("Please provide a valid server address with the -serverAddr flag")
		return
	}

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
			logger.Error("failed to start server", "error", err)
		}
	}()

	rpcClient, err := ethclient.Dial(*rpcAddr)
	if err != nil {
		logger.Error("failed to connect to rpc", "error", err)
		return
	}

	// Since we don't know if the server has TLS enabled on its rpc
	// endpoint, we try different strategies from most secure to
	// least secure. In the future, when only TLS-enabled servers
	// are allowed, only the TLS system pool certificate strategy
	// should be used.
	var conn *grpc.ClientConn
	for _, e := range []struct {
		strategy   string
		isSecure   bool
		credential credentials.TransportCredentials
	}{
		{"TLS system pool certificate", true, credentials.NewClientTLSFromCert(nil, "")},
		{"TLS skip verification", false, credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})},
		{"TLS disabled", false, insecure.NewCredentials()},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		logger.Info("dialing to grpc server", "strategy", e.strategy)
		conn, err = grpc.DialContext(
			ctx,
			*serverAddr,
			grpc.WithBlock(),
			grpc.WithTransportCredentials(e.credential),
		)
		if err != nil {
			logger.Error("failed to dial grpc server", "error", err)
			cancel()
			continue
		}

		cancel()
		if !e.isSecure {
			logger.Warn("established connection with the grpc server has potential security risk")
		}
		break
	}
	if conn == nil {
		logger.Error("dialing of grpc server failed")
		return
	}
	defer conn.Close()

	bidderClient := pb.NewBidderClient(conn)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			err = checkOrPrepay(bidderClient, logger)
			if err != nil {
				logger.Error("failed to check or stake", "error", err)
			}
			<-ticker.C
		}
	}()

	for i := 0; i < *parallelWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				err = sendBid(bidderClient, logger, rpcClient)
				if err != nil {
					logger.Error("failed to send bid", "error", err)
				}
				time.Sleep(1 * time.Second)
			}
		}()
	}

	wg.Wait()
}

func checkOrPrepay(
	bidderClient pb.BidderClient,
	logger *slog.Logger,
) error {
	allowance, err := bidderClient.GetAllowance(context.Background(), &pb.EmptyMessage{})
	if err != nil {
		logger.Error("failed to get allowance", "error", err)
		return err
	}

	logger.Info("prepaid allowance", "amount", allowance.Amount)

	minAllowance, err := bidderClient.GetMinAllowance(context.Background(), &pb.EmptyMessage{})
	if err != nil {
		logger.Error("failed to get min allowance", "error", err)
		return err
	}

	allowanceAmt, set := big.NewInt(0).SetString(allowance.Amount, 10)
	if !set {
		logger.Error("failed to parse allowance amount")
		return errors.New("failed to parse allowance amount")
	}

	minAllowanceAmt, set := big.NewInt(0).SetString(minAllowance.Amount, 10)
	if !set {
		logger.Error("failed to parse min allowance amount")
		return errors.New("failed to parse min allowance amount")
	}

	if allowanceAmt.Cmp(minAllowanceAmt) > 0 {
		logger.Error("bidder already has balance")
		return nil
	}

	topup := big.NewInt(0).Mul(minAllowanceAmt, big.NewInt(10))

	_, err = bidderClient.PrepayAllowance(context.Background(), &pb.PrepayRequest{
		Amount: topup.String(),
	})
	if err != nil {
		logger.Error("failed to prepay allowance", "error", err)
		return err
	}

	logger.Info("prepaid allowance", "amount", topup.String())

	return nil
}

func sendBid(
	bidderClient pb.BidderClient,
	logger *slog.Logger,
	rpcClient *ethclient.Client,
) error {
	blkNum, err := rpcClient.BlockNumber(context.Background())
	if err != nil {
		logger.Error("failed to get block number", "error", err)
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
		TxHashes:            []string{strings.TrimPrefix(txHash.Hex(), "0x")},
		Amount:              strconv.Itoa(int(amount)),
		BlockNumber:         int64(blkNum) + 5,
		DecayStartTimestamp: time.Now().UnixMilli() - (time.Duration(8 * time.Second).Milliseconds()),
		DecayEndTimestamp:   time.Now().UnixMilli(),
	}

	logger.Info("sending bid", "bid", bid)

	start := time.Now()
	rcv, err := bidderClient.SendBid(context.Background(), bid)
	if err != nil {
		logger.Error("failed to send bid", "error", err)
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
