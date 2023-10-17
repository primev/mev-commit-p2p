package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	configFile string
)

func main() {
	logger := logrus.New()
	var rootCmd = &cobra.Command{Use: "searcher-cli"}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config-file", "c", "config.yaml", "Configuration file path")

	if len(os.Args) > 1 && os.Args[1] != "create-config" {
		viper.SetConfigFile(configFile)
		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err != nil {
			logger.Error(err)
			return
		}

		serverAddress = viper.GetString("serverAddress")
		logLevel = viper.GetString("logLevel")
		logOutput = viper.GetString("logOutput")

		logger.Infof("Using config file: %s", configFile)
	} else {
		logLevel = "debug"
		logOutput = "stdout"
	}

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
		logFile, err := os.OpenFile("searcher-cli.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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

	// NOTE: (@iowar) By sending an empty Bid request, the status of the RPC
	// server is being checked. Instead, a ping request can be defined within
	// the searcher proto or a better solution can be found. Seeking the team's
	// opinion on this
	statusServerCmd := &cobra.Command{
		Use:   "status",
		Short: "Check the status of the gRPC searcher server",
		Run: func(cmd *cobra.Command, args []string) {
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

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)
			defer cancel()

			_, err = client.SendBid(ctx, &pb.Bid{})
			if err != nil {
				logger.Warnf("gRPC searcher server is not reachable at %s", serverAddress)
				return
			}

			logger.Infof("gRPC searcher server is up and running at %s", serverAddress)
		},
	}

	createConfigCmd := &cobra.Command{
		Use:   "create-config",
		Short: "Create an example config.yaml file",
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat("config.yaml")
			if err == nil {
				logger.Warn("config.yaml file already exists in the current directory.")
				return
			}

			configData := `serverAddress: "localhost:13524"
logLevel: "debug"
logOutput: "stdout"`
			err = createConfigFile("config.yaml", []byte(configData))
			if err != nil {
				logger.Error(err)
			} else {
				logger.Info("config.yaml file created in the current directory")
			}
		},
	}

	rootCmd.AddCommand(sendBidCmd)
	rootCmd.AddCommand(statusServerCmd)
	rootCmd.AddCommand(createConfigCmd)

	if err := rootCmd.Execute(); err != nil {
		return
	}
}

func createConfigFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return err
	}
	return nil
}
