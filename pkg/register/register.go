package register

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Register is the provider register used to query the contract
type Register interface {
	// GetStake returns stake of specified provider
	GetStake(provider common.Address) (*big.Int, error)
}
