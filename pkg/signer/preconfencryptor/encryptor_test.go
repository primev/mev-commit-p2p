package preconfencryptor_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper"
	mockkeysigner "github.com/primevprotocol/mev-commit/pkg/keykeeper/keysigner/mock"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfencryptor"
	"github.com/stretchr/testify/assert"
)

func TestBids(t *testing.T) {
	t.Parallel()

	t.Run("bid", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		address := crypto.PubkeyToAddress(key.PublicKey)
		keySigner := mockkeysigner.NewMockKeySigner(key, address)
		keyKeeper, err := keykeeper.NewBidderKeyKeeper(keySigner)
		if err != nil {
			t.Fatal(err)
		}
		encryptor := preconfencryptor.NewEncryptor(keyKeeper)

		start := time.Now().UnixMilli()
		end := start + 100000
		_, encryptedBid, err := encryptor.ConstructEncryptedBid("0xkartik", "10", 2, start, end)
		if err != nil {
			t.Fatal(err)
		}

		providerKeyKeeper, err := keykeeper.NewProviderKeyKeeper(keySigner)
		if err != nil {
			t.Fatal(err)
		}
		providerKeyKeeper.SetAESKey(address, keyKeeper.AESKey)
		encryptorProvider := preconfencryptor.NewEncryptor(providerKeyKeeper)
		bid, err := encryptorProvider.DecryptBidData(address, encryptedBid)
		if err != nil {
			t.Fatal(err)
		}

		bidAddress, err := encryptor.VerifyBid(bid)
		if err != nil {
			t.Fatal(err)
		}

		originatorAddress, pubkey, err := encryptor.BidOriginator(bid)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, address, *originatorAddress)
		assert.Equal(t, address, *bidAddress)
		assert.Equal(t, key.PublicKey, *pubkey)
	})
	t.Run("preConfirmation", func(t *testing.T) {
		bidderKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		keySigner := mockkeysigner.NewMockKeySigner(bidderKey, crypto.PubkeyToAddress(bidderKey.PublicKey))
		bidderKeyKeeper, err := keykeeper.NewBidderKeyKeeper(keySigner)
		if err != nil {
			t.Fatal(err)
		}
		bidderEncryptor := preconfencryptor.NewEncryptor(bidderKeyKeeper)
		providerKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}

		bidderAddress := crypto.PubkeyToAddress(bidderKey.PublicKey)
		keySigner = mockkeysigner.NewMockKeySigner(providerKey, crypto.PubkeyToAddress(providerKey.PublicKey))
		providerKeyKeeper, err := keykeeper.NewProviderKeyKeeper(keySigner)
		if err != nil {
			t.Fatal(err)
		}

		providerKeyKeeper.SetAESKey(bidderAddress, bidderKeyKeeper.AESKey)
		providerEncryptor := preconfencryptor.NewEncryptor(providerKeyKeeper)
		start := time.Now().UnixMilli()
		end := start + 100000

		bid, encryptedBid, err := bidderEncryptor.ConstructEncryptedBid("0xkartik", "10", 2, start, end)
		if err != nil {
			t.Fatal(err)
		}

		decryptedBid, err := providerEncryptor.DecryptBidData(bidderAddress, encryptedBid)
		if err != nil {
			t.Fatal(err)
		}
		_, encryptedPreConfirmation, err := providerEncryptor.ConstructEncryptedPreConfirmation(decryptedBid)
		if err != nil {
			t.Fail()
		}

		_, address, err := bidderEncryptor.VerifyEncryptedPreConfirmation(providerKeyKeeper.GetNIKEPublicKey(), bid.Digest, encryptedPreConfirmation)
		if err != nil {
			t.Fail()
		}

		assert.Equal(t, crypto.PubkeyToAddress(providerKey.PublicKey), *address)
	})
}

func TestHashing(t *testing.T) {
	t.Parallel()

	t.Run("bid", func(t *testing.T) {
		bid := &preconfpb.Bid{
			TxHash:              "0xkartik",
			BidAmount:           "200",
			BlockNumber:         3000,
			DecayStartTimestamp: 10,
			DecayEndTimestamp:   30,
		}

		hash, err := preconfencryptor.GetBidHash(bid)
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
		bidHash := "a0327970258c49b922969af74d60299a648c50f69a2d98d6ab43f32f64ac2100"
		bidSignature := "876c1216c232828be9fabb14981c8788cebdf6ed66e563c4a2ccc82a577d052543207aeeb158a32d8977736797ae250c63ef69a82cd85b727da21e20d030fb311b"

		bidHashBytes, err := hex.DecodeString(bidHash)
		if err != nil {
			t.Fatal(err)
		}
		bidSigBytes, err := hex.DecodeString(bidSignature)
		if err != nil {
			t.Fatal(err)
		}

		bid := &preconfpb.Bid{
			TxHash:              "0xkartik",
			BidAmount:           "2",
			BlockNumber:         2,
			DecayStartTimestamp: 10,
			DecayEndTimestamp:   20,
			Digest:              bidHashBytes,
			Signature:           bidSigBytes,
		}

		preConfirmation := &preconfpb.PreConfirmation{
			Bid: bid,
		}

		hash, err := preconfencryptor.GetPreConfirmationHash(preConfirmation)
		if err != nil {
			t.Fatal(err)
		}

		hashStr := hex.EncodeToString(hash)
		expHash := "7492710e0487466ee0cd9f795ce1bb72e1b17ebe6d7b0bb729f2a65a8e756f9b"
		if hashStr != expHash {
			t.Fatalf("hash mismatch: %s != %s", hashStr, expHash)
		}
	})
}

// todo: mock key encryptor
// func TestSignature(t *testing.T) {
// 	t.Parallel()
// 	// alice keys 0x328809Bc894f92807417D2dAD6b7C998c1aFdac6
// 	pkey, err := crypto.HexToECDSA("9C0257114EB9399A2985F8E75DAD7600C5D89FE3824FFA99EC1C3EB8BF3B0501")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	keySigner := mockkeysigner.NewMockKeySigner(pkey, crypto.PubkeyToAddress(pkey.PublicKey))
// 	bidderKK, err := keykeeper.NewBidderKeyKeeper(keySigner)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	bidder := preconfencryptor.NewEncryptor(bidderKK)

// 	bid, _, err := bidder.ConstructEncryptedBid("0xkartik", "2", 2, 10, 20)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// bob keys 0x1D96F2f6BeF1202E4Ce1Ff6Dad0c2CB002861d3e
// 	providerKey, err := crypto.HexToECDSA("38E47A7B719DCE63662AEAF43440326F551B8A7EE198CEE35CB5D517F2D296A2")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	keySigner = mockkeysigner.NewMockKeySigner(providerKey, crypto.PubkeyToAddress(providerKey.PublicKey))
// 	providerKK, err := keykeeper.NewProviderKeyKeeper(keySigner)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	provider := preconfencryptor.NewEncryptor(providerKK)
// 	preconf, err := provider.ConstructEncryptedPreConfirmation(bid)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// expCommitmentDigest := "1a85d596184bc14bcc974fcc276430f5295e24d8a6e09cda91a2fd2f72257d29"
// 	// expCommitmentSig := "9a7a3b1a0cbbc50e4bea5991f0005db26cd5d20f80595df433b837039b20a4e62078edb2390085d1a01aad16fddb9eea0b2f641433c946c9ab1708db7d5065c91b"
// 	// if hex.EncodeToString(preconf.Commitment) != expCommitmentDigest {
// 	// 	t.Fatalf("digest mismatch: %s != %s", hex.EncodeToString(preconf.Commitment), expCommitmentDigest)
// 	// }
// 	// if hex.EncodeToString(preconf.Signature) != expCommitmentSig {
// 		// t.Fatalf("signature mismatch: %s != %s", hex.EncodeToString(preconf.Signature), expCommitmentSig)
// 	}
// }

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

	owner, err := preconfencryptor.EIPVerify(bidHashBytes, bidHashBytes, bidSigBytes)
	if err != nil {
		t.Fatal(err)
	}

	expOwner := "0x8339F9E3d7B2693aD8955Aa5EC59D56669A84d60"
	if owner.Hex() != expOwner {
		t.Fatalf("owner mismatch: %s != %s", owner.Hex(), expOwner)
	}
}
