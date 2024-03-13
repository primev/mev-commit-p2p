package keysigner

import (
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type privateKeySigner struct {
	privKey *ecdsa.PrivateKey
}

func NewPrivateKeySigner(wr io.Writer, path string) (*privateKeySigner, error) {
	privKeyFile, err := resolveFilePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key file path: %w", err)
	}

	if err := createKeyIfNotExists(wr, privKeyFile); err != nil {
		return nil, fmt.Errorf("failed to create private key: %w", err)
	}

	privKey, err := crypto.LoadECDSA(privKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from file '%s': %w", privKeyFile, err)
	}

	return &privateKeySigner{
		privKey: privKey,
	}, nil
}

func (pks *privateKeySigner) SignHash(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, pks.privKey)
}

func (pks *privateKeySigner) SignTx(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return types.SignTx(tx, types.NewLondonSigner(chainID), pks.privKey)
}

func (pks *privateKeySigner) GetAddress() common.Address {
	return crypto.PubkeyToAddress(pks.privKey.PublicKey)
}

func (pks *privateKeySigner) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return pks.privKey, nil
}

// ZeroPrivateKey does nothing because the private key for PKS persists in memory
// and should not be deleted.
func (pks *privateKeySigner) ZeroPrivateKey(key *ecdsa.PrivateKey) {}

func extractPrivateKey(keystoreFile, passphrase string) (*ecdsa.PrivateKey, error) {
	keyjson, err := os.ReadFile(keystoreFile)
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(keyjson, passphrase)
	if err != nil {
		return nil, err
	}

	// Overwrite the keyjson slice with zeros to wipe the sensitive data from memory.
	// This is a security measure to reduce the risk of the encrypted key being extracted from memory.
	for i := range keyjson {
		keyjson[i] = 0
	}

	return key.PrivateKey, nil
}

func createKeyIfNotExists(wr io.Writer, path string) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(wr, "using existing private key:", path)
		return nil
	}

	fmt.Fprintln(wr, "creating new private key:", path)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	if err := crypto.SaveECDSA(path, key); err != nil {
		return err
	}

	addr := crypto.PubkeyToAddress(key.PublicKey)

	fmt.Fprintln(wr, "private key saved to file:", path)
	fmt.Fprintln(wr, "wallet address:", addr.Hex())

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
