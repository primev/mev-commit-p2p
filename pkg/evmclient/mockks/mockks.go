package mockks

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
)

type mockKeyStore struct {
}

func NewMockKeyStore() *mockKeyStore {
	return &mockKeyStore{}
}

func (m *mockKeyStore) SignTxWithPassphrase(a accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return tx, nil
}
