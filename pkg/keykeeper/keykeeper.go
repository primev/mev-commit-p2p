package keykeeper

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper/keysigner"
)

func NewBidderKeyKeeper(keysigner keysigner.KeySigner) (*BidderKeyKeeper, error) {
	aesKey, err := generateAESKey()
	if err != nil {
		return nil, err
	}

	bidHashesToNIKE := make(map[string]*ecdh.PrivateKey)

	return &BidderKeyKeeper{
		KeySigner:       keysigner,
		AESKey:          aesKey,
		BidHashesToNIKE: bidHashesToNIKE,
	}, nil
}

func (bkk *BidderKeyKeeper) SignHash(data []byte) ([]byte, error) {
	return bkk.KeySigner.SignHash(data)
}

func (bkk *BidderKeyKeeper) GetAddress() common.Address {
	return bkk.KeySigner.GetAddress()
}

func (bkk *BidderKeyKeeper) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return bkk.KeySigner.GetPrivateKey()
}

func (bkk *BidderKeyKeeper) ZeroPrivateKey(key *ecdsa.PrivateKey) {
	bkk.KeySigner.ZeroPrivateKey(key)
}

func (bkk *BidderKeyKeeper) GenerateNIKEKeys(bidHash []byte) (*ecdh.PublicKey, error) {
	nikePrivateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	nikePublicKey := nikePrivateKey.PublicKey()
	bkk.BidHashesToNIKE[hex.EncodeToString(bidHash)] = nikePrivateKey
	return nikePublicKey, nil
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

func (pkk *ProviderKeyKeeper) SignHash(data []byte) ([]byte, error) {
	return pkk.KeySigner.SignHash(data)
}

func (pkk *ProviderKeyKeeper) GetAddress() common.Address {
	return pkk.KeySigner.GetAddress()
}

func (pkk *ProviderKeyKeeper) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return pkk.KeySigner.GetPrivateKey()
}

func (pkk *ProviderKeyKeeper) ZeroPrivateKey(key *ecdsa.PrivateKey) {
	pkk.KeySigner.ZeroPrivateKey(key)
}

func (pkk *ProviderKeyKeeper) GetNIKEPrivateKey() *ecdh.PrivateKey {
	return pkk.keys.NIKEPrivateKey
}

func NewBootnodeKeyKeeper(keysigner keysigner.KeySigner) *BootnodeKeyKeeper {
	return &BootnodeKeyKeeper{
		KeySigner: keysigner,
	}
}

func (btkk *BootnodeKeyKeeper) SignHash(data []byte) ([]byte, error) {
	return btkk.KeySigner.SignHash(data)
}

func (btkk *BootnodeKeyKeeper) GetAddress() common.Address {
	return btkk.KeySigner.GetAddress()
}

func (btkk *BootnodeKeyKeeper) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	return btkk.KeySigner.GetPrivateKey()
}

func (btkk *BootnodeKeyKeeper) ZeroPrivateKey(key *ecdsa.PrivateKey) {
	btkk.KeySigner.ZeroPrivateKey(key)
}
