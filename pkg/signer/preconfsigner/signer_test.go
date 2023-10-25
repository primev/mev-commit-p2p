package preconfsigner_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"github.com/stretchr/testify/assert"
)

func TestBids(t *testing.T) {
	t.Parallel()

	t.Run("bid", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		signer := preconfsigner.NewSigner(key)

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
	t.Run("preConfirmation", func(t *testing.T) {
		userKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		userSigner := preconfsigner.NewSigner(userKey)

		providerKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		providerSigner := preconfsigner.NewSigner(providerKey)

		bid, err := userSigner.ConstructSignedBid("0xkartik", big.NewInt(10), big.NewInt(2))
		if err != nil {
			t.Fatal(err)
		}

		preConfirmation, err := providerSigner.ConstructPreConfirmation(bid)
		if err != nil {
			t.Fail()
		}

		address, err := userSigner.VerifyPreConfirmation(preConfirmation)
		if err != nil {
			t.Fail()
		}

		assert.Equal(t, crypto.PubkeyToAddress(providerKey.PublicKey), *address)
	})
}
