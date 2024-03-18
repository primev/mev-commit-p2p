package keykeeper

import (
	"crypto/ecdh"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper/keysigner"
)

type KeyKeeper interface {
	SignHash(data []byte) ([]byte, error)
	GetAddress() common.Address
	GetPrivateKey() (*ecdsa.PrivateKey, error)
	ZeroPrivateKey(key *ecdsa.PrivateKey)
}

type ProviderKeys struct {
	EncryptionPrivateKey *ecies.PrivateKey
	EncryptionPublicKey  *ecies.PublicKey
	NIKEPrivateKey       *ecdh.PrivateKey
	NIKEPublicKey        *ecdh.PublicKey
}

type ProviderKeyKeeper struct {
	keys           ProviderKeys
	KeySigner      keysigner.KeySigner
	BiddersAESKeys map[common.Address][]byte
}

type BidderKeyKeeper struct {
	AESKey          []byte
	KeySigner       keysigner.KeySigner
	BidHashesToNIKE map[string]*ecdh.PrivateKey
}

type BootnodeKeyKeeper struct {
	KeySigner keysigner.KeySigner
}
