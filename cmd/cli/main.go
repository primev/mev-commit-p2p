package main

import (
	"context"
	"os"
	"strings"

	pb "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddress string
	txHash        string
	amount        int64
	blockNumber   int64
	logLevel      string
	logOutput     string
)

func main() {
	var rootCmd = &cobra.Command{Use: "searcher-cli"}
	rootCmd.PersistentFlags().StringVarP(&serverAddress, "server", "s", "localhost:13524", "gRPC searcher server address")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "debug", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVarP(&logOutput, "log-output", "o", "stdout", "Log output (stdout, file)")

	logger := logrus.New()

	switch strings.ToLower(logLevel) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.DebugLevel)
	}

	if logOutput == "stdout" {
		logger.SetOutput(os.Stdout)
	} else {
		logFile, err := os.Create("mev-commit.log")
		if err != nil {
			logger.Fatal("Error opening log file", err)
		}
		logger.SetOutput(logFile)
	}

	sendBidCmd := &cobra.Command{
		Use:   "send-bid",
		Short: "Send a bid to the gRPC server",
		Run: func(cmd *cobra.Command, args []string) {
			if txHash == "" || amount == 0 || blockNumber == 0 {
				logger.WithFields(logrus.Fields{
					"txHash":      txHash,
					"amount":      amount,
					"blockNumber": blockNumber,
				}).Warn("Missing required arguments. Please provide --txhash, --amount, and --block.")
				return
			}

			creds := insecure.NewCredentials()
			conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(creds))
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err,
				}).Error("Connection error")
				return
			}
			defer conn.Close()

			client := pb.NewSearcherClient(conn)

			bid := &pb.Bid{
				TxHash:      txHash,
				Amount:      amount,
				BlockNumber: blockNumber,
			}

			ctx := context.Background()
			stream, err := client.SendBid(ctx, bid)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err,
				}).Error("SendBid call failed")
				return
			}

			preConfirmation, err := stream.Recv()
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err,
				}).Error("PreConfirmation not received")
				return
			}

			logger.WithFields(logrus.Fields{
				"info": "received preconfirmation",
			}).Info(preConfirmation)
		},
	}

	sendBidCmd.Flags().StringVarP(&txHash, "txhash", "t", "", "Transaction hash")
	sendBidCmd.Flags().Int64VarP(&amount, "amount", "a", 0, "Bid amount")
	sendBidCmd.Flags().Int64VarP(&blockNumber, "block", "b", 0, "Block number")

	rootCmd.AddCommand(sendBidCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("CLI error")
	}
}
