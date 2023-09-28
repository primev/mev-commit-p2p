package register

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Register is the provider register used to query the contract
type Register interface {
	// GetMinimumStake returns the minimum stake required to be a provider
	GetMinimumStake() (*big.Int, error)
	// GetStake returns stake of specified provider
	GetStake(provider common.Address) (*big.Int, error)
}

type register struct{}

func New() Register {
	return &register{}
}

func (r *register) GetMinimumStake() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (r *register) GetStake(provider common.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}
