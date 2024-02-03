package mockkeysigner

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type mockKeySigner struct {
	privKey *ecdsa.PrivateKey
	address common.Address
}

func NewMockKeySigner(privKey *ecdsa.PrivateKey, address common.Address) *mockKeySigner {
	return &mockKeySigner{privKey: privKey, address: address}
}

func (m *mockKeySigner) SignTx(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return tx, nil
}

func (m *mockKeySigner) SignHash(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, m.privKey)
}

func (m *mockKeySigner) GetAddress() common.Address {
	return m.address
}

func (m *mockKeySigner) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return m.privKey, nil
}
