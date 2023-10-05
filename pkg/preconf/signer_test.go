package preconf_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/preconf"
	"github.com/stretchr/testify/assert"
)

func TestBids(t *testing.T) {
	t.Parallel()

	t.Run("bid", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		signer := preconf.NewSigner(key)

		bid, err := signer.ConstructSignedBid("0xkartik", big.NewInt(10), big.NewInt(2))
		if err != nil {
			t.Fatal(err)
		}

		address, err := signer.VerifyBid(bid)
		if err != nil {
			t.Fatal(err)
		}

		expectedAddress := crypto.PubkeyToAddress(key.PublicKey)

		originatorAddress, pubkey, err := signer.BidOriginator(bid)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expectedAddress, *originatorAddress)
		assert.Equal(t, expectedAddress, *address)
		assert.Equal(t, key.PublicKey, *pubkey)
	})
	t.Run("commitment", func(t *testing.T) {
		userKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		userSigner := preconf.NewSigner(userKey)

		builderKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		builderSigner := preconf.NewSigner(builderKey)

		bid, err := userSigner.ConstructSignedBid("0xkartik", big.NewInt(10), big.NewInt(2))
		if err != nil {
			t.Fatal(err)
		}

		commitment, err := builderSigner.ConstructCommitment(bid)
		if err != nil {
			t.Fail()
		}

		address, err := userSigner.VerifyCommitment(commitment)
		if err != nil {
			t.Fail()
		}

		assert.Equal(t, crypto.PubkeyToAddress(builderKey.PublicKey), *address)
	})
}
