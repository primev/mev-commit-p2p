package mockks

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

type mockKeyStore struct {
	privateKey *ecdsa.PrivateKey
}

func NewMockKeyStore(privateKey *ecdsa.PrivateKey) *mockKeyStore {
	return &mockKeyStore{privateKey: privateKey}
}

func (m *mockKeyStore) SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	return crypto.Sign(hash, m.privateKey)
}
