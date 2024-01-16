package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

const (
	defaultKeyFile = "key"
)

var (
	optionConfigDir = &cli.StringFlag{
		Name:    "directory",
		Aliases: []string{"dir"},
		Usage:   "The directory to store config files",
		EnvVars: []string{"MEV_COMMIT_CONFIG_DIR"},
		Value:   defaultConfigDir,
	}

	optionPeerType = &cli.StringFlag{
		Name:    "peer-type",
		Usage:   "The type of peer to run. Options are 'bidder' or 'provider'",
		EnvVars: []string{"MEV_COMMIT_PEER_TYPE"},
		Value:   "bidder",
	}

	optionRPCEndpoint = &cli.StringFlag{
		Name:    "rpc-endpoint",
		Usage:   "The Settlement chain RPC endpoint to connect to",
		EnvVars: []string{"MEV_COMMIT_RPC_ENDPOINT"},
		Value:   "http://localhost:8545",
	}
)

func initNode(c *cli.Context) error {
	rpcEndpoint := c.String(optionRPCEndpoint.Name)

	dir, err := resolveFilePath(c.String(optionConfigDir.Name))
	if err != nil {
		return fmt.Errorf("failed to resolve config directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// check if config file exists
	configFile := filepath.Join(dir, defaultConfigFile)
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	keyFile := filepath.Join(dir, defaultKeyFile)
	err = createKeyAt(c, keyFile)
	if err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	peerType := c.String(optionPeerType.Name)
	if peerType != "bidder" && peerType != "provider" {
		return fmt.Errorf("invalid peer type: %s", peerType)
	}

	// create config file
	conf := &config{
		PrivKeyFile: keyFile,
		Secret:      defaultSecret,
		P2PPort:     defaultP2PPort,
		HTTPPort:    defaultHTTPPort,
		RPCPort:     defaultRPCPort,
		PeerType:    peerType,
		RPCEndpoint: rpcEndpoint,
		Bootnodes:   defaultBootnodes,
		LogLevel:    "info",
		LogFmt:      "text",
	}

	buf, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, buf, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created config file: %s\n", configFile)
	return nil
}
