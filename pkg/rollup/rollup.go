package rollup

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Rollup interface {
	// GetBuilderAddress returns current builder address
	GetBuilderAddress() common.Address

	// GetSubscriptionEnd returns subscription end for specified commitment
	GetSubscriptionEnd(commitment common.Hash) (*big.Int, error)

	// GetMinimalStake returns minimal stake of specified builder
	GetMinimalStake(builder common.Address) (*big.Int, error)

	// GetCommitment calculates commitment hash for this builder by searcher address
	GetCommitment(searcher common.Address) common.Hash

	// GetBlockNumber returns latest blocks number from rollup
	GetBlockNumber() (*big.Int, error)
}
