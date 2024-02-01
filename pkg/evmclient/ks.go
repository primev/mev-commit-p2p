package evmclient

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
)

// KeyStorer is an interface for mocking KeyStore
type KeyStoreEVMClient interface {
	SignTxWithPassphrase(a accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}
