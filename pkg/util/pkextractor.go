package util

import (
	"crypto/ecdsa"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type KeyExtractor interface {
	ExtractPrivateKey(keystoreFile, passphrase string) (*ecdsa.PrivateKey, error)
}

type keyExtractor struct{}

func NewKeyExtractor() *keyExtractor {
	return &keyExtractor{}
}

func (ke *keyExtractor) ExtractPrivateKey(keystoreFile, passphrase string) (*ecdsa.PrivateKey, error) {
	keyjson, err := os.ReadFile(keystoreFile)
	if err != nil {
		return nil, err
	}

	key, err := keystore.DecryptKey(keyjson, passphrase)
	if err != nil {
		return nil, err
	}

	// Zero out the keyjson slice
	for i := range keyjson {
		keyjson[i] = 0
	}

	return key.PrivateKey, nil
}
