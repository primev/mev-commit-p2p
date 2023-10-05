package preconf

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (p *privateKeySigner) BidOriginator(bid *Bid) (*common.Address, *ecdsa.PublicKey, error) {
	_, err := p.VerifyBid(bid)
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := crypto.SigToPub(bid.BidHash, bid.Signature)
	if err != nil {
		return nil, nil, err
	}

	address := crypto.PubkeyToAddress(*pubkey)

	return &address, pubkey, nil
}

func (p *privateKeySigner) CommitmentOriginator(
	c *Commitment,
) (*common.Address, *ecdsa.PublicKey, error) {
	_, err := p.VerifyCommitment(c)
	if err != nil {
		return nil, nil, err
	}

	pubkey, err := crypto.SigToPub(c.DataHash, c.CommitmentSignature)
	if err != nil {
		return nil, nil, err
	}

	address := crypto.PubkeyToAddress(*pubkey)

	return &address, pubkey, nil
}
