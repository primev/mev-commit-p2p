package signer

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

type Signer interface {
	// create signature
	Sign(*ecdsa.PrivateKey, []byte) ([]byte, error)
	// verify signature and return signer address
	Verify([]byte, []byte) (bool, common.Address, error)
}
