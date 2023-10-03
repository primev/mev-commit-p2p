package preconf_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/structures/preconf"
	"github.com/stretchr/testify/assert"
)

func TestBids(t *testing.T) {
	t.Parallel()

	t.Run("bid", func(t *testing.T) {
		key, _ := crypto.GenerateKey()

		bid, err := preconf.ConstructSignedBid(big.NewInt(10), "0xkartik", big.NewInt(2), preconf.PrivateKeySigner{PrivKey: key})
		if err != nil {
			t.Fatal(err)
		}
		address, err := bid.VerifySearcherSignature()

		if err != nil {
			t.Fatal(err)
		}

		expectedAddress := crypto.PubkeyToAddress(key.PublicKey)

		originatorAddress, pubkey, err := bid.BidOriginator()
		if err != nil {
			t.Fail()
		}
		assert.Equal(t, expectedAddress, *originatorAddress)
		assert.Equal(t, expectedAddress, *address)
		assert.Equal(t, key.PublicKey, *pubkey)
	})
	t.Run("commitment", func(t *testing.T) {
		userKey, _ := crypto.GenerateKey()
		userSigner := preconf.PrivateKeySigner{PrivKey: userKey}

		builderKey, _ := crypto.GenerateKey()
		builderSigner := preconf.PrivateKeySigner{PrivKey: builderKey}

		bid, err := preconf.ConstructSignedBid(big.NewInt(10), "0xkartik", big.NewInt(2), userSigner)
		if err != nil {
			t.Fatal(err)
		}
		commitment, err := preconf.ConstructCommitment(*bid, builderSigner)
		if err != nil {
			t.Fail()
		}
		address, err := commitment.VerifyBuilderSignature()
		if err != nil {
			t.Fail()
		}

		assert.Equal(t, crypto.PubkeyToAddress(builderKey.PublicKey), *address)

	})
}
