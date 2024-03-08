package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	contracts "github.com/primevprotocol/contracts-abi/config"
	mevcommit "github.com/primevprotocol/mev-commit"
	ks "github.com/primevprotocol/mev-commit/pkg/keysigner"
	"github.com/primevprotocol/mev-commit/pkg/node"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const (
	defaultP2PAddr = "0.0.0.0"
	defaultP2PPort = 13522

	defaultHTTPPort = 13523
	defaultRPCPort  = 13524

	defaultConfigDir = "~/.mev-commit"
	defaultKeyFile   = "key"
	defaultSecret    = "secret"
	defaultKeystore  = "keystore"
)

var (
	portCheck = func(c *cli.Context, p int) error {
		if p < 0 || p > 65535 {
			return fmt.Errorf("invalid port number %d, expected 0 <= port <= 65535", p)
		}
		return nil
	}

	stringInCheck = func(flag string, opts []string) func(c *cli.Context, s string) error {
		return func(c *cli.Context, s string) error {
			if !slices.Contains(opts, s) {
				return fmt.Errorf("invalid %s option %q, expected one of %s", flag, s, strings.Join(opts, ", "))
			}
			return nil
		}
	}

	addressPortCheck = func(c *cli.Context, s string) error {
		host, port, err := net.SplitHostPort(s)
		if err != nil {
			return fmt.Errorf("invalid value %q: %w", s, err)
		}

		p, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("invalid value %q: invalid port: %w", s, err)
		}

		if err := portCheck(c, p); err != nil {
			return fmt.Errorf("invalid value %q: invalid port: %w", s, err)
		}

		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("invalid value %q: incorrect ip address", s)
		}
		return nil
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

	optionKeystorePassword = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "keystore-password",
		Usage:   "use to access keystore",
		EnvVars: []string{"MEV_COMMIT_KEYSTORE_PASSWORD"},
	})

	optionKeystorePath = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "keystore-path",
		Usage:   "path to keystore location",
		EnvVars: []string{"MEV_COMMIT_KEYSTORE_PATH"},
		Value:   filepath.Join(defaultConfigDir, defaultKeystore),
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

	optionP2PAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "p2p-addr",
		Usage:   "address to bind for p2p connections",
		EnvVars: []string{"MEV_COMMIT_P2P_ADDR"},
		Value:   defaultP2PAddr,
	})

	optionHTTPPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:    "http-port",
		Usage:   "port to listen for http connections",
		EnvVars: []string{"MEV_COMMIT_HTTP_PORT"},
		Value:   defaultHTTPPort,
		Action:  portCheck,
	})

	optionHTTPAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "http-addr",
		Usage:   "address to bind for http connections",
		EnvVars: []string{"MEV_COMMIT_HTTP_ADDR"},
		Value:   "",
	})

	optionRPCPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:    "rpc-port",
		Usage:   "port to listen for rpc connections",
		EnvVars: []string{"MEV_COMMIT_RPC_PORT"},
		Value:   defaultRPCPort,
		Action:  portCheck,
	})

	optionRPCAddr = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "rpc-addr",
		Usage:   "address to bind for RPC connections",
		EnvVars: []string{"MEV_COMMIT_RPC_ADDR"},
		Value:   "",
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

	optionLogSinkTCP = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "log-sink-tcp",
		Usage:   "log sink tcp configures the logger to also send log messages to the specified <address:port> using the TCP protocol",
		EnvVars: []string{"MEV_COMMIT_LOG_SINK_TCP"},
		Action:  addressPortCheck,
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

	optionNATPort = altsrc.NewIntFlag(&cli.IntFlag{
		Name:    "nat-port",
		Usage:   "externally mapped port for the node",
		EnvVars: []string{"MEV_COMMIT_NAT_PORT"},
		Value:   defaultP2PPort,
	})

	optionServerTLSCert = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "server-tls-certificate",
		Usage:   "Path to the server TLS certificate",
		EnvVars: []string{"MEV_COMMIT_SERVER_TLS_CERTIFICATE"},
	})

	optionServerTLSPrivateKey = altsrc.NewStringFlag(&cli.StringFlag{
		Name:    "server-tls-private-key",
		Usage:   "Path to the server TLS private key",
		EnvVars: []string{"MEV_COMMIT_SERVER_TLS_PRIVATE_KEY"},
	})
)

func main() {
	flags := []cli.Flag{
		optionConfig,
		optionPeerType,
		optionPrivKeyFile,
		optionKeystorePassword,
		optionKeystorePath,
		optionP2PPort,
		optionP2PAddr,
		optionHTTPPort,
		optionHTTPAddr,
		optionRPCPort,
		optionRPCAddr,
		optionBootnodes,
		optionSecret,
		optionLogFmt,
		optionLogLevel,
		optionLogSinkTCP,
		optionBidderRegistryAddr,
		optionProviderRegistryAddr,
		optionPreconfStoreAddr,
		optionSettlementRPCEndpoint,
		optionNATAddr,
		optionNATPort,
		optionServerTLSCert,
		optionServerTLSPrivateKey,
	}

	app := &cli.App{
		Name:    "mev-commit",
		Usage:   "Start mev-commit node",
		Version: mevcommit.Version(),
		Flags:   flags,
		Before:  altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc(optionConfig.Name)),
		Action:  initializeApplication,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(app.Writer, "exited with error:", err)
	}
}

func initializeApplication(c *cli.Context) error {
	if c.IsSet(optionLogSinkTCP.Name) {
		if err := addTCPSink(c); err != nil {
			return err
		}
	}
	if err := verifyKeystorePasswordPresence(c); err != nil {
		return err
	}
	if err := launchNodeWithConfig(c); err != nil {
		return err
	}
	return nil
}

// addTCPSink configures the cli.App.Writer and cli.App.ErrWriter to also send
// log messages to the specified address and port using the TCP protocol.
func addTCPSink(c *cli.Context) error {
	conn, err := net.Dial("tcp", c.String(optionLogSinkTCP.Name))
	if err != nil {
		return fmt.Errorf("failed to connect to TCP server: %w", err)
	}
	c.App.Writer = io.MultiWriter(c.App.Writer, conn)
	c.App.ErrWriter = io.MultiWriter(c.App.ErrWriter, conn)
	return nil
}

// verifyKeystorePasswordPresence checks for the presence of a keystore password.
// it returns error, if keystore path is set and keystore password is not
func verifyKeystorePasswordPresence(c *cli.Context) error {
	if c.IsSet(optionKeystorePath.Name) && !c.IsSet(optionKeystorePassword.Name) {
		return cli.Exit("Password for encrypted keystore is missing", 1)
	}
	return nil
}

// launchNodeWithConfig configures and starts the p2p node based on the CLI context.
func launchNodeWithConfig(c *cli.Context) error {
	keysigner, err := newKeySigner(c)
	if err != nil {
		return fmt.Errorf("failed to create keysigner: %w", err)
	}

	logger, err := newLogger(
		c.String(optionLogLevel.Name),
		c.String(optionLogFmt.Name),
		c.App.Writer,
	)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	httpAddr := fmt.Sprintf("%s:%d", c.String(optionHTTPAddr.Name), c.Int(optionHTTPPort.Name))
	rpcAddr := fmt.Sprintf("%s:%d", c.String(optionRPCAddr.Name), c.Int(optionRPCPort.Name))
	natAddr := ""
	if c.String(optionNATAddr.Name) != "" {
		natAddr = fmt.Sprintf("%s:%d", c.String(optionNATAddr.Name), c.Int(optionNATPort.Name))
	}

	crtFile := c.String(optionServerTLSCert.Name)
	keyFile := c.String(optionServerTLSPrivateKey.Name)
	if (crtFile == "") != (keyFile == "") {
		return fmt.Errorf("both -%s and -%s must be provided to enable TLS", optionServerTLSCert.Name, optionServerTLSPrivateKey.Name)
	}

	nd, err := node.NewNode(&node.Options{
		KeySigner:                keysigner,
		Secret:                   c.String(optionSecret.Name),
		PeerType:                 c.String(optionPeerType.Name),
		P2PPort:                  c.Int(optionP2PPort.Name),
		P2PAddr:                  c.String(optionP2PAddr.Name),
		HTTPAddr:                 httpAddr,
		RPCAddr:                  rpcAddr,
		Logger:                   logger,
		Bootnodes:                c.StringSlice(optionBootnodes.Name),
		PreconfContract:          c.String(optionPreconfStoreAddr.Name),
		ProviderRegistryContract: c.String(optionProviderRegistryAddr.Name),
		BidderRegistryContract:   c.String(optionBidderRegistryAddr.Name),
		RPCEndpoint:              c.String(optionSettlementRPCEndpoint.Name),
		NatAddr:                  natAddr,
		TLSCertificateFile:       crtFile,
		TLSPrivateKeyFile:        keyFile,
	})
	if err != nil {
		return fmt.Errorf("failed starting node: %w", err)
	}

	<-c.Done()
	fmt.Fprintln(c.App.Writer, "shutting down...")
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

	// We check if the logger was writing
	// to a TCP stream and close it properly.
	if conn, ok := c.App.Writer.(net.Conn); ok {
		if err := conn.Close(); err != nil {
			return fmt.Errorf("faild to close TCP connection: %w", err)
		}
	}

	return nil
}

func newLogger(lvl, logFmt string, sink io.Writer) (*slog.Logger, error) {
	level := new(slog.LevelVar)
	if err := level.UnmarshalText([]byte(lvl)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	var (
		handler slog.Handler
		options = &slog.HandlerOptions{
			AddSource: true,
			Level:     level,
		}
	)
	switch logFmt {
	case "text":
		handler = slog.NewTextHandler(sink, options)
	case "json", "none":
		handler = slog.NewJSONHandler(sink, options)
	default:
		return nil, fmt.Errorf("invalid log format: %s", logFmt)
	}

	return slog.New(handler), nil
}

func newKeySigner(c *cli.Context) (ks.KeySigner, error) {
	if c.IsSet(optionKeystorePath.Name) {
		return ks.NewKeystoreSigner(c.App.Writer, c.String(optionKeystorePath.Name), c.String(optionKeystorePassword.Name))
	}
	return ks.NewPrivateKeySigner(c.App.Writer, c.String(optionPrivKeyFile.Name))
}
