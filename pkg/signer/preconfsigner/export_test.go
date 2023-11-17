package preconfsigner

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var EIPVerify = eipVerify

func (p *privateKeySigner) BidOriginator(bid *Bid) (*common.Address, *ecdsa.PublicKey, error) {
	_, err := p.VerifyBid(bid)
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := crypto.SigToPub(bid.Digest, bid.Signature)
	if err != nil {
		return nil, nil, err
	}

	address := crypto.PubkeyToAddress(*pubkey)

	return &address, pubkey, nil
}

func (p *privateKeySigner) PreConfirmationOriginator(
	c *PreConfirmation,
) (*common.Address, *ecdsa.PublicKey, error) {
	_, err := p.VerifyPreConfirmation(c)
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := crypto.SigToPub(c.Digest, c.Signature)
	if err != nil {
		return nil, nil, err
	}

	address := crypto.PubkeyToAddress(*pubkey)

	return &address, pubkey, nil
}
