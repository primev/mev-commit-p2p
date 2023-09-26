package registermock

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/register"
)

type mockRegister struct {
	stake int64
}

func New(stake int64) register.Register {
	return &mockRegister{stake: stake}
}

func (t *mockRegister) GetStake(_ common.Address) (*big.Int, error) {
	return big.NewInt(t.stake), nil
}
