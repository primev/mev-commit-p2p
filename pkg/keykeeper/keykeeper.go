package keykeeper

import (
	"crypto/ecdh"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper/keysigner"
)

func NewBidderKeyKeeper(keysigner keysigner.KeySigner) (*BidderKeyKeeper, error) {
	aesKey, err := generateAESKey()
	if err != nil {
		return nil, err
	}

	return &BidderKeyKeeper{
		KeySigner: keysigner,
		AESKey:    aesKey,
	}, nil
}

func NewProviderKeyKeeper(keysigner keysigner.KeySigner) (*ProviderKeyKeeper, error) {
	biddersAESKeys := make(map[common.Address][]byte)

	encryptionPrivateKey, err := ecies.GenerateKey(rand.Reader, elliptic.P256(), nil)
	if err != nil {
		return nil, err
	}

	nikePrivateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &ProviderKeyKeeper{
		KeySigner:      keysigner,
		BiddersAESKeys: biddersAESKeys,
		keys: ProviderKeys{
			EncryptionPrivateKey: encryptionPrivateKey,
			EncryptionPublicKey:  &encryptionPrivateKey.PublicKey,
			NIKEPrivateKey:       nikePrivateKey,
			NIKEPublicKey:        nikePrivateKey.PublicKey(),
		},
	}, nil
}

func (pkk *ProviderKeyKeeper) GetNIKEPublicKey() *ecdh.PublicKey {
	return pkk.keys.NIKEPublicKey
}

func (pkk *ProviderKeyKeeper) GetECIESPublicKey() *ecies.PublicKey {
	return pkk.keys.EncryptionPublicKey
}

func (pkk *ProviderKeyKeeper) DecryptWithECIES(message []byte) ([]byte, error) {
	return pkk.keys.EncryptionPrivateKey.Decrypt(message, nil, nil)
}
