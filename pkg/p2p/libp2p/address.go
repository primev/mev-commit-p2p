package libp2p

import (
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

	pbDcomBytes := elliptic.Marshal(secp256k1.S256(), pbDcom.X, pbDcom.Y)

	hash := sha3.NewLegacyKeccak256()
	hash.Write(pbDcomBytes[1:])
	address := hash.Sum(nil)[12:]

	return common.BytesToAddress(address), nil
}
