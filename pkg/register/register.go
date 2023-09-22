package register

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Register is the provider register used to query the contract
type Register interface {
	// GetMinimalStake returns minimal stake of specified provider
	GetMinimalStake(provider common.Address) (*big.Int, error)
}
