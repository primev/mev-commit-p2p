package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	contracts "github.com/primevprotocol/contracts-abi/config"
	mevcommit "github.com/primevprotocol/mev-commit"
	"github.com/primevprotocol/mev-commit/pkg/node"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

const (
	defaultP2PPort  = 13522
	defaultHTTPPort = 13523
	defaultRPCPort  = 13524

	defaultConfigDir  = "~/.mev-commit"
	defaultConfigFile = "config.yaml"
	defaultSecret     = "secret"
)

var (
	defaultBootnodes = []string{
		"/ip4/69.67.151.95/tcp/13522/p2p/16Uiu2HAmLYUvthfDCewNMdfPhrVefBbsfaPL22fWWfC2zuoh5SpV",
	}
)

var (
	optionConfig = &cli.StringFlag{
		Name:     "config",
		Usage:    "path to config file",
		Required: true,
		EnvVars:  []string{"MEV_COMMIT_CONFIG"},
		Value:    filepath.Join(defaultConfigDir, defaultConfigFile),
	}
)

func main() {
	app := &cli.App{
		Name:    "mev-commit",
		Usage:   "Entry point for mev-commit",
		Version: mevcommit.Version(),
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize a mev-commit node",
				Flags: []cli.Flag{
					optionConfigDir,
					optionRPCEndpoint,
					optionPeerType,
				},
				Action: initNode,
			},
			{
				Name:  "start",
				Usage: "Start the mev-commit node",
				Flags: []cli.Flag{
					optionConfig,
				},
				Action: start,
			},
			{
				Name:      "create-key",
				Usage:     "Create a new ECDSA private key and save it to a file",
				ArgsUsage: "<output_file>",
				Action:    createKey,
			},
		}}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(app.Writer, "exited with error: %v\n", err)
	}
}

func createKey(c *cli.Context) error {
	if len(c.Args().Slice()) != 1 {
		return fmt.Errorf("usage: mev-commit create-key <output_file>")
	}

	outputFile := c.Args().Slice()[0]

	return createKeyAt(c, outputFile)
}

func createKeyAt(c *cli.Context, path string) error {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	if err := crypto.SaveECDSA(path, privKey); err != nil {
		return err
	}

	wallet := libp2p.GetEthAddressFromPubKey(&privKey.PublicKey)

	fmt.Fprintf(c.App.Writer, "Private key saved to file: %s\n", path)
	fmt.Fprintf(c.App.Writer, "Wallet address: %s\n", wallet.Hex())
	return nil
}

type config struct {
	PrivKeyFile              string   `yaml:"priv_key_file" json:"priv_key_file"`
	Secret                   string   `yaml:"secret" json:"secret"`
	PeerType                 string   `yaml:"peer_type" json:"peer_type"`
	P2PPort                  int      `yaml:"p2p_port" json:"p2p_port"`
	HTTPPort                 int      `yaml:"http_port" json:"http_port"`
	RPCPort                  int      `yaml:"rpc_port" json:"rpc_port"`
	LogFmt                   string   `yaml:"log_fmt" json:"log_fmt"`
	LogLevel                 string   `yaml:"log_level" json:"log_level"`
	Bootnodes                []string `yaml:"bootnodes" json:"bootnodes"`
	PreconfContract          string   `yaml:"preconf_contract" json:"preconf_contract"`
	ProviderRegistryContract string   `yaml:"provider_registry_contract" json:"provider_registry_contract"`
	BidderRegistryContract   string   `yaml:"bidder_registry_contract" json:"bidder_registry_contract"`
	RPCEndpoint              string   `yaml:"rpc_endpoint" json:"rpc_endpoint"`
	NatAddr                  string   `yaml:"nat_addr" json:"nat_addr"`
}

func checkConfig(cfg *config) error {
	if cfg.PrivKeyFile == "" {
		return fmt.Errorf("priv_key_file is required")
	}

	if cfg.PeerType == "" {
		return fmt.Errorf("peer_type is required")
	}

	if cfg.RPCEndpoint == "" {
		return fmt.Errorf("rpc_endpoint is required")
	}

	if cfg.Secret == "" {
		cfg.Secret = "welcome"
	}

	if cfg.P2PPort == 0 {
		cfg.P2PPort = defaultP2PPort
	}

	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = defaultHTTPPort
	}

	if cfg.RPCPort == 0 {
		cfg.RPCPort = defaultRPCPort
	}

	if cfg.LogFmt == "" {
		cfg.LogFmt = "text"
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	if cfg.BidderRegistryContract == "" {
		cfg.BidderRegistryContract = contracts.TestnetContracts.BidderRegistry
	}

	if cfg.ProviderRegistryContract == "" {
		cfg.ProviderRegistryContract = contracts.TestnetContracts.ProviderRegistry
	}

	if cfg.PreconfContract == "" && cfg.PeerType == "provider" {
		cfg.PreconfContract = contracts.TestnetContracts.PreconfCommitmentStore
	}

	if cfg.Bootnodes == nil {
		cfg.Bootnodes = defaultBootnodes
	}

	return nil
}

func resolveFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		return filepath.Join(home, path[1:]), nil
	}

	return path, nil
}

func start(c *cli.Context) error {
	configFile, err := resolveFilePath(c.String(optionConfig.Name))
	if err != nil {
		return fmt.Errorf("failed to resolve config file path: %w", err)
	}

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

	privKeyFile, err := resolveFilePath(cfg.PrivKeyFile)
	if err != nil {
		return fmt.Errorf("failed to resolve private key file path: %w", err)
	}

	privKey, err := crypto.LoadECDSA(privKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load private key from file '%s': %w", cfg.PrivKeyFile, err)
	}

	nd, err := node.NewNode(&node.Options{
		PrivKey:                  privKey,
		Secret:                   cfg.Secret,
		PeerType:                 cfg.PeerType,
		P2PPort:                  cfg.P2PPort,
		HTTPPort:                 cfg.HTTPPort,
		RPCPort:                  cfg.RPCPort,
		Logger:                   logger,
		Bootnodes:                cfg.Bootnodes,
		PreconfContract:          cfg.PreconfContract,
		ProviderRegistryContract: cfg.ProviderRegistryContract,
		BidderRegistryContract:   cfg.BidderRegistryContract,
		RPCEndpoint:              cfg.RPCEndpoint,
		NatAddr:                  cfg.NatAddr,
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
		handler = slog.NewTextHandler(sink, &slog.HandlerOptions{AddSource: true, Level: level})
	case "none":
		fallthrough
	case "json":
		handler = slog.NewJSONHandler(sink, &slog.HandlerOptions{AddSource: true, Level: level})
	default:
		return nil, fmt.Errorf("invalid log format: %s", logFmt)
	}

	return slog.New(handler), nil
}
