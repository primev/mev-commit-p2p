package preconfsigner_test

import (
	"encoding/hex"
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

func TestHashing(t *testing.T) {
	t.Parallel()

	t.Run("bid", func(t *testing.T) {
		bid := &preconfsigner.Bid{
			TxHash:      "0xkartik",
			BidAmt:      big.NewInt(2),
			BlockNumber: big.NewInt(2),
		}

		hash, err := preconfsigner.GetBidHash(bid)
		if err != nil {
			t.Fatal(err)
		}

		hashStr := hex.EncodeToString(hash)
		expHash := "86ac45fb1e987a6c8115494cd4fd82f6756d359022cdf5ea19fd2fac1df6e7f0"
		if hashStr != expHash {
			t.Fatalf("hash mismatch: %s != %s", hashStr, expHash)
		}
	})

	t.Run("preConfirmation", func(t *testing.T) {
		bidHash := "86ac45fb1e987a6c8115494cd4fd82f6756d359022cdf5ea19fd2fac1df6e7f0"
		bidSignature := "33683da4605067c9491d665864b2e4e7ade8bc57921da9f192a1b8246a941eaa2fb90f72031a2bf6008fa590158591bb5218c9aace78ad8cf4d1f2f4d74bc3e901"

		bidHashBytes, err := hex.DecodeString(bidHash)
		if err != nil {
			t.Fatal(err)
		}
		bidSigBytes, err := hex.DecodeString(bidSignature)
		if err != nil {
			t.Fatal(err)
		}

		bid := &preconfsigner.Bid{
			TxHash:      "0xkartik",
			BidAmt:      big.NewInt(2),
			BlockNumber: big.NewInt(2),
			Digest:      bidHashBytes,
			Signature:   bidSigBytes,
		}

		preConfirmation := &preconfsigner.PreConfirmation{
			Bid: *bid,
		}

		hash, err := preconfsigner.GetPreConfirmationHash(preConfirmation)
		if err != nil {
			t.Fatal(err)
		}

		hashStr := hex.EncodeToString(hash)
		expHash := "31dca6c6fd15593559dabb9e25285f727fd33f07e17ec2e8da266706020034dc"
		if hashStr != expHash {
			t.Fatalf("hash mismatch: %s != %s", hashStr, expHash)
		}
	})
}
