package preconfsigner_test

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	mockkeysigner "github.com/primevprotocol/mev-commit/pkg/keysigner/mock"
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

		keySigner := mockkeysigner.NewMockKeySigner(key, crypto.PubkeyToAddress(key.PublicKey))
		signer := preconfsigner.NewSigner(keySigner)

		bid, err := signer.ConstructSignedBid("0xkartik", big.NewInt(10), big.NewInt(2), uint64(time.Now().UnixMilli()), uint64(time.Now().UnixMilli()))
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
		bidderKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		keySigner := mockkeysigner.NewMockKeySigner(bidderKey, crypto.PubkeyToAddress(bidderKey.PublicKey))

		bidderSigner := preconfsigner.NewSigner(keySigner)
		providerKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		keySigner = mockkeysigner.NewMockKeySigner(providerKey, crypto.PubkeyToAddress(providerKey.PublicKey))
		providerSigner := preconfsigner.NewSigner(keySigner)

		bid, err := bidderSigner.ConstructSignedBid("0xkartik", big.NewInt(10), big.NewInt(2), 1, 2)
		if err != nil {
			t.Fatal(err)
		}

		preConfirmation, err := providerSigner.ConstructPreConfirmation(bid)
		if err != nil {
			t.Fail()
		}

		address, err := bidderSigner.VerifyPreConfirmation(preConfirmation)
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
			TxHash:              "0xkartik",
			BidAmt:              big.NewInt(200),
			BlockNumber:         big.NewInt(3000),
			DecayStartTimeStamp: 10,
			DecayEndTimeStamp:   30,
		}

		hash, err := preconfsigner.GetBidHash(bid)
		if err != nil {
			t.Fatal(err)
		}

		hashStr := hex.EncodeToString(hash)
		// This hash is sourced from the solidity contract to ensure interoperability
		expHash := "a837b0c680d4b9b11011ac6225670498d845e65f1dc340b00694d74a6ca0a049"
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

func TestVerify(t *testing.T) {
	t.Parallel()

	bidSig := "8af22e36247e14ba05d3a5a3cc62eee708cfd9ce293c0aebcbe7f89229f6db56638af8427806247d9abb295f681c1a2f2bb127f3bf80799f80d62b252cce04d91c"
	bidHash := "2574b1ab8a90e173528ddee748be8e8e696b1f0cf687f75966550f5e9ef408b0"

	bidHashBytes, err := hex.DecodeString(bidHash)
	if err != nil {
		t.Fatal(err)
	}

	bidSigBytes, err := hex.DecodeString(bidSig)
	if err != nil {
		t.Fatal(err)
	}

	// Adjust the last byte if it's 27 or 28
	if bidSigBytes[64] >= 27 && bidSigBytes[64] <= 28 {
		bidSigBytes[64] -= 27
	}

	owner, err := preconfsigner.EIPVerify(bidHashBytes, bidHashBytes, bidSigBytes)
	if err != nil {
		t.Fatal(err)
	}

	expOwner := "0x8339F9E3d7B2693aD8955Aa5EC59D56669A84d60"
	if owner.Hex() != expOwner {
		t.Fatalf("owner mismatch: %s != %s", owner.Hex(), expOwner)
	}
}
