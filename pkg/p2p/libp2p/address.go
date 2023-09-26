package libp2p

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	core "github.com/libp2p/go-libp2p/core"
	"golang.org/x/crypto/sha3"
)

func GetEthAddressFromPeerID(peerID core.PeerID) (common.Address, error) {
	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return common.Address{}, err
	}

	pubKeyBytes, err := pubKey.Raw()
	if err != nil {
		return common.Address{}, err
	}

	pbDcom, err := crypto.DecompressPubkey(pubKeyBytes)
	if err != nil {
		return common.Address{}, err
	}

	return GetEthAddressFromPubKey(pbDcom), nil
}

func GetEthAddressFromPubKey(key *ecdsa.PublicKey) common.Address {
	pbBytes := elliptic.Marshal(secp256k1.S256(), key.X, key.Y)

	hash := sha3.NewLegacyKeccak256()
	hash.Write(pbBytes[1:])
	address := hash.Sum(nil)[12:]

	return common.BytesToAddress(address)
}
