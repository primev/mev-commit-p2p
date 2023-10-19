package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	pb "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"
)

var (
	txHash      string
	amount      int64
	blockNumber int64
)

type config struct {
	ServerAddress string `json:"server_address" yaml:"server_address"`
	LogFmt        string `json:"log_fmt" yaml:"log_fmt"`
	LogLevel      string `json:"log_level" yaml:"log_level"`
}

var (
	optionConfig = &cli.StringFlag{
		Name:     "config",
		Usage:    "path to config file",
		Required: true,
		EnvVars:  []string{"SEARCHER_CLI_CONFIG"},
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "searcher-cli"
	app.Usage = "A CLI tool for interacting with a gRPC searcher server"
	app.Version = "1.0.0"

	var (
		cfg    config
		logger *slog.Logger
	)

	app.Flags = []cli.Flag{
		optionConfig,
	}

	app.Before = func(c *cli.Context) error {
		configFile := c.String(optionConfig.Name)
		fmt.Printf("using configuration file: %s\n", configFile)

		buf, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file at '%s': %w", configFile, err)
		}

		if err := yaml.Unmarshal(buf, &cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config file at '%s': %w", configFile, err)
		}

		if err := checkConfig(&cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config file at '%s': %w", configFile, err)
		}

		logger, err = newLogger(cfg.LogLevel, cfg.LogFmt, c.App.Writer)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:  "send-bid",
			Usage: "Send a bid to the gRPC server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "txhash",
					Usage:       "Transaction hash",
					Destination: &txHash,
				},
				&cli.Int64Flag{
					Name:        "amount",
					Usage:       "Bid amount",
					Destination: &amount,
				},
				&cli.Int64Flag{
					Name:        "block",
					Usage:       "Block number",
					Destination: &blockNumber,
				},
			},
			Action: func(c *cli.Context) error {
				if txHash == "" || amount == 0 || blockNumber == 0 {
					return fmt.Errorf("Missing required arguments. Please provide --txhash, --amount, and --block.")
				}

				creds := insecure.NewCredentials()
				conn, err := grpc.Dial(cfg.ServerAddress, grpc.WithTransportCredentials(creds))
				if err != nil {
					return err
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
					return err
				}

				preConfirmation, err := stream.Recv()
				if err != nil {
					return err
				}

				logger.Info("received preconfirmation", "preconfirmation", preConfirmation)
				return nil
			},
		},
		{
			// NOTE: (@iowar) By sending an empty Bid request, the status of the RPC
			// server is being checked. Instead, a ping request can be defined within
			// the searcher proto or a better solution can be found. Seeking the team's
			// opinion on this
			Name:  "status",
			Usage: "Check the status of the gRPC searcher server",
			Action: func(c *cli.Context) error {
				creds := insecure.NewCredentials()
				conn, err := grpc.Dial(cfg.ServerAddress, grpc.WithTransportCredentials(creds))
				if err != nil {
					return err
				}
				defer conn.Close()

				client := pb.NewSearcherClient(conn)

				ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)
				defer cancel()

				_, err = client.SendBid(ctx, &pb.Bid{})
				if err != nil {
					logger.Info("gRPC searcher server is not reachable", "server", cfg.ServerAddress)
					return nil
				}

				logger.Info("gRPC searcher server is up and running", "server", cfg.ServerAddress)
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(app.Writer, "exited with error: %v\n", err)
	}
}

func checkConfig(cfg *config) error {
	if cfg.ServerAddress == "" {
		return fmt.Errorf("server_address is required")
	}

	if cfg.LogFmt == "" {
		cfg.LogFmt = "text"
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return nil
}

func newLogger(lvl, logFmt string, sink io.Writer) (*slog.Logger, error) {
	var (
		level   = new(slog.LevelVar)
		handler slog.Handler
	)

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
		return nil, fmt.Errorf("invalid log level: %s", lvl)
	}

	switch logFmt {
	case "text":
		handler = slog.NewTextHandler(sink, &slog.HandlerOptions{Level: level})
	case "none":
		fallthrough
	case "json":
		handler = slog.NewJSONHandler(sink, &slog.HandlerOptions{Level: level})
	default:
		return nil, fmt.Errorf("invalid log format: %s", logFmt)
	}

	return slog.New(handler), nil
}
