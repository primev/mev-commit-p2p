package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/node"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

const (
	defaultP2PPort  = 13522
	defaultHTTPPort = 13523
)

var (
	optionConfig = &cli.StringFlag{
		Name:     "config",
		Usage:    "path to config file",
		Required: true,
		EnvVars:  []string{"MEV_COMMIT_CONFIG"},
	}
)

func main() {
	app := &cli.App{
		Name:  "mev-commit",
		Usage: "Entry point for mev-commit",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start mev-commit",
				Flags: []cli.Flag{
					optionConfig,
				},
				Action: func(c *cli.Context) error {
					return start(c)
				},
			},
			{
				Name: "create-key",
				Action: func(c *cli.Context) error {
					return createKey(c)
				},
			},
		}}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(app.Writer, "exited with error: %v\n", err)
	}
}

func createKey(c *cli.Context) error {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	if len(c.Args().Slice()) != 1 {
		return fmt.Errorf("usage: mev-commit create-key <output_file>")
	}

	outputFile := c.Args().Slice()[0]

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	defer f.Close()

	if err := crypto.SaveECDSA(outputFile, privKey); err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Private key saved to file: %s\n", outputFile)
	return nil
}

type config struct {
	PrivKeyFile string   `yaml:"priv_key_file" json:"priv_key_file"`
	Secret      string   `yaml:"secret" json:"secret"`
	PeerType    string   `yaml:"peer_type" json:"peer_type"`
	P2PPort     int      `yaml:"p2p_port" json:"p2p_port"`
	HTTPPort    int      `yaml:"http_port" json:"http_port"`
	LogFmt      string   `yaml:"log_fmt" json:"log_fmt"`
	LogLevel    string   `yaml:"log_level" json:"log_level"`
	Bootnodes   []string `yaml:"bootnodes" json:"bootnodes"`
}

func checkConfig(cfg *config) error {
	if cfg.PrivKeyFile == "" {
		return fmt.Errorf("priv_key_file is required")
	}

	if cfg.Secret == "" {
		return fmt.Errorf("secret is required")
	}

	if cfg.PeerType == "" {
		return fmt.Errorf("peer_type is required")
	}

	if cfg.P2PPort == 0 {
		cfg.P2PPort = defaultP2PPort
	}

	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = defaultHTTPPort
	}

	if cfg.LogFmt == "" {
		cfg.LogFmt = "text"
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return nil
}

func start(c *cli.Context) error {
	configFile := c.String(optionConfig.Name)
	fmt.Fprintf(c.App.Writer, "starting mev-commit with config file: %s\n", configFile)

	var cfg config
	buf, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file at '%s': %w", configFile, err)
	}

	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config file at '%s': %w", configFile, err)
	}

	if err := checkConfig(&cfg); err != nil {
		return fmt.Errorf("invalid config file at '%s': %w", configFile, err)
	}

	logger, err := newLogger(cfg.LogLevel, cfg.LogFmt, c.App.Writer)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	privKey, err := crypto.LoadECDSA(cfg.PrivKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load private key from file '%s': %w", cfg.PrivKeyFile, err)
	}

	nd, err := node.NewNode(&node.Options{
		PrivKey:   privKey,
		Secret:    cfg.Secret,
		PeerType:  cfg.PeerType,
		P2PPort:   cfg.P2PPort,
		HTTPPort:  cfg.HTTPPort,
		Logger:    logger,
		Bootnodes: cfg.Bootnodes,
	})
	if err != nil {
		return fmt.Errorf("failed starting node: %w", err)
	}

	<-c.Done()
	fmt.Fprintf(c.App.Writer, "shutting down...\n")
	closed := make(chan struct{})

	go func() {
		defer close(closed)

		err := nd.Close()
		if err != nil {
			logger.Error("failed to close node", "error", err)
		}
	}()

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		logger.Error("failed to close node in time")
	}

	return nil
}

func newLogger(lvl, logFmt string, sink io.Writer) (*slog.Logger, error) {
	var (
		level   = new(slog.LevelVar) // Info by default
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
