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
	"github.com/urfave/cli/v2/altsrc"
)

const (
	defaultP2PPort  = 13522
	defaultHTTPPort = 13523
	defaultRPCPort  = 13524

	defaultConfigDir = "~/.mev-commit"
	defaultKeyFile   = "key"
	defaultSecret    = "secret"
)

var (
	portCheck = func(c *cli.Context, p int) error {
		if p < 0 || p > 65535 {
			return fmt.Errorf("Invalid port number %d, expected 0 <= port <= 65535", p)
		}
		return nil
	}

	stringInCheck = func(flag string, opts []string) func(c *cli.Context, p string) error {
		return func(c *cli.Context, p string) error {
			for _, opt := range opts {
				if p == opt {
					return nil
				}
			}
			return fmt.Errorf("Invalid %s option '%s', expected one of %s", flag, p, strings.Join(opts, ", "))
		}
	}
)

var (
	optionConfig = &cli.StringFlag{
		Name:    "config",
		Usage:   "path to config file",
		EnvVars: []string{"MEV_COMMIT_CONFIG"},
	}

	optionPrivKeyFile = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "priv-key-file",
		Usage:   "path to private key file",
		EnvVars: []string{"MEV_COMMIT_PRIVKEY_FILE"},
		Value:   filepath.Join(defaultConfigDir, defaultKeyFile),
	})

	optionPeerType = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "peer-type",
		Usage:   "peer type to use, options are 'bidder', 'provider' or 'bootnode'",
		EnvVars: []string{"MEV_COMMIT_PEER_TYPE"},
		Value:   "bidder",
		Action:  stringInCheck("peer-type", []string{"bidder", "provider", "bootnode"}),
	})

	optionP2PPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:    "p2p-port",
		Usage:   "port to listen for p2p connections",
		EnvVars: []string{"MEV_COMMIT_P2P_PORT"},
		Value:   defaultP2PPort,
		Action:  portCheck,
	})

	optionHTTPPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:    "http-port",
		Usage:   "port to listen for http connections",
		EnvVars: []string{"MEV_COMMIT_HTTP_PORT"},
		Value:   defaultHTTPPort,
		Action:  portCheck,
	})

	optionRPCPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:    "rpc-port",
		Usage:   "port to listen for rpc connections",
		EnvVars: []string{"MEV_COMMIT_RPC_PORT"},
		Value:   defaultRPCPort,
		Action:  portCheck,
	})

	optionBootnodes = altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
		Name:    "bootnodes",
		Usage:   "list of bootnodes to connect to",
		EnvVars: []string{"MEV_COMMIT_BOOTNODES"},
	})

	optionSecret = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "secret",
		Usage:   "secret to use for signing",
		EnvVars: []string{"MEV_COMMIT_SECRET"},
		Value:   defaultSecret,
	})

	optionLogFmt = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "log-fmt",
		Usage:   "log format to use, options are 'text' or 'json'",
		EnvVars: []string{"MEV_COMMIT_LOG_FMT"},
		Value:   "text",
		Action:  stringInCheck("log-fmt", []string{"text", "json"}),
	})

	optionLogLevel = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "log-level",
		Usage:   "log level to use, options are 'debug', 'info', 'warn', 'error'",
		EnvVars: []string{"MEV_COMMIT_LOG_LEVEL"},
		Value:   "info",
		Action:  stringInCheck("log-level", []string{"debug", "info", "warn", "error"}),
	})

	optionBidderRegistryAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "bidder-registry-contract",
		Usage:   "address of the bidder registry contract",
		EnvVars: []string{"MEV_COMMIT_BIDDER_REGISTRY_ADDR"},
		Value:   contracts.TestnetContracts.BidderRegistry,
	})

	optionProviderRegistryAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "provider-registry-contract",
		Usage:   "address of the provider registry contract",
		EnvVars: []string{"MEV_COMMIT_PROVIDER_REGISTRY_ADDR"},
		Value:   contracts.TestnetContracts.ProviderRegistry,
	})

	optionPreconfStoreAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "preconf-contract",
		Usage:   "address of the preconfirmation commitment store contract",
		EnvVars: []string{"MEV_COMMIT_PRECONF_ADDR"},
		Value:   contracts.TestnetContracts.PreconfCommitmentStore,
	})

	optionSettlementRPCEndpoint = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "settlement-rpc-endpoint",
		Usage:   "rpc endpoint of the settlement layer",
		EnvVars: []string{"MEV_COMMIT_SETTLEMENT_RPC_ENDPOINT"},
		Value:   "http://localhost:8545",
	})

	optionNATAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "nat-addr",
		Usage:   "external address of the node",
		EnvVars: []string{"MEV_COMMIT_NAT_ADDR"},
	})
)

func main() {
	app := &cli.App{
		Name:    "mev-commit",
		Usage:   "Start mev-commit node",
		Version: mevcommit.Version(),
		Flags: []cli.Flag{
			optionConfig,
			optionPeerType,
			optionPrivKeyFile,
			optionP2PPort,
			optionHTTPPort,
			optionRPCPort,
			optionBootnodes,
			optionSecret,
			optionLogFmt,
			optionLogLevel,
			optionBidderRegistryAddr,
			optionProviderRegistryAddr,
			optionPreconfStoreAddr,
			optionSettlementRPCEndpoint,
			optionNATAddr,
		},
		Action: start,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(app.Writer, "exited with error: %v\n", err)
	}
}

func createKeyIfNotExists(c *cli.Context, path string) error {
	// check if key already exists
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(c.App.Writer, "Using existing private key: %s\n", path)
		return nil
	}

	fmt.Fprintf(c.App.Writer, "Creating new private key: %s\n", path)

	// check if parent directory exists
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		// create parent directory
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}

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
	privKeyFile, err := resolveFilePath(c.String(optionPrivKeyFile.Name))
	if err != nil {
		return fmt.Errorf("failed to get private key file path: %w", err)
	}

	if err := createKeyIfNotExists(c, privKeyFile); err != nil {
		return fmt.Errorf("failed to create private key: %w", err)
	}

	logger, err := newLogger(
		c.String(optionLogLevel.Name),
		c.String(optionLogFmt.Name),
		c.App.Writer,
	)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	privKey, err := crypto.LoadECDSA(privKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load private key from file '%s': %w", privKeyFile, err)
	}

	nd, err := node.NewNode(&node.Options{
		PrivKey:                  privKey,
		Secret:                   c.String(optionSecret.Name),
		PeerType:                 c.String(optionPeerType.Name),
		P2PPort:                  c.Int(optionP2PPort.Name),
		HTTPPort:                 c.Int(optionHTTPPort.Name),
		RPCPort:                  c.Int(optionRPCPort.Name),
		Logger:                   logger,
		Bootnodes:                c.StringSlice(optionBootnodes.Name),
		PreconfContract:          c.String(optionPreconfStoreAddr.Name),
		ProviderRegistryContract: c.String(optionProviderRegistryAddr.Name),
		BidderRegistryContract:   c.String(optionBidderRegistryAddr.Name),
		RPCEndpoint:              c.String(optionSettlementRPCEndpoint.Name),
		NatAddr:                  c.String(optionNATAddr.Name),
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
