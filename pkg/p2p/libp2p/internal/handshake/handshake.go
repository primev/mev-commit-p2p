package handshake

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/register"
	"github.com/primevprotocol/mev-commit/pkg/signer"
	"golang.org/x/crypto/sha3"
)

const (
	ProtocolName    = "handshake"
	ProtocolVersion = "1.0.0"
	StreamName      = "handshake"
)

// Handshake is the handshake protocol
type Service struct {
	privKey      *ecdsa.PrivateKey
	ethAddress   common.Address
	peerType     p2p.PeerType
	passcode     string
	signer       signer.Signer
	register     register.Register
	minimumStake *big.Int
}

func ProtocolID() protocol.ID {
	return protocol.ID(p2p.NewStreamName(
		ProtocolName,
		ProtocolVersion,
		StreamName,
	))
}

type HandshakeReq struct {
	PeerType string
	Token    string
	Sig      []byte
}

type HandshakeResp struct {
	ObservedAddress common.Address
	PeerType        string
	Ack             *HandshakeReq
}

func getEthAddressFromPeerID(peerID core.PeerID) (common.Address, error) {
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

func (h *Service) verifySignature(
	req *HandshakeReq,
	peerID core.PeerID,
) (common.Address, error) {
	unsignedData := []byte(req.PeerType + req.Token)

	verified, ethAddress, err := h.signer.Verify(req.Sig, unsignedData)
	if err != nil {
		return common.Address{}, err
	}

	if !verified {
		return common.Address{}, errors.New("signature verification failed")
	}

	observedEthAddress, err := getEthAddressFromPeerID(peerID)
	if err != nil {
		return common.Address{}, err
	}

	if !bytes.Equal(observedEthAddress.Bytes(), ethAddress.Bytes()) {
		return common.Address{}, errors.New("observed address mismatch")
	}

	return ethAddress, nil
}

func (h *Service) createSignature() ([]byte, error) {
	unsignedData := []byte(h.peerType.String() + h.passcode)
	sig, err := h.signer.Sign(h.privKey, unsignedData)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func (h *Service) Handle(
	ctx context.Context,
	stream p2p.Stream,
	peerID core.PeerID,
) (p2p.Peer, error) {

	r, w := msgpack.NewReaderWriter[HandshakeReq, HandshakeResp](stream)
	req, err := r.ReadMsg(ctx)
	if err != nil {
		return p2p.Peer{}, err
	}

	ethAddress, err := h.verifySignature(req, peerID)
	if err != nil {
		return p2p.Peer{}, err
	}

	if req.PeerType == "builder" {
		stake, err := h.register.GetMinimalStake(ethAddress)
		if err != nil {
			return p2p.Peer{}, err
		}

		if stake.Cmp(h.minimumStake) < 0 {
			return p2p.Peer{}, errors.New("stake insufficient")
		}
	}

	sig, err := h.createSignature()
	if err != nil {
		return p2p.Peer{}, err
	}

	resp := &HandshakeResp{
		ObservedAddress: ethAddress,
		PeerType:        req.PeerType,
		Ack: &HandshakeReq{
			PeerType: h.peerType.String(),
			Token:    h.passcode,
			Sig:      sig,
		},
	}

	if err := w.WriteMsg(ctx, resp); err != nil {
		return p2p.Peer{}, err
	}

	return p2p.Peer{
		EthAddress: ethAddress,
		Type:       p2p.FromString(req.PeerType),
	}, nil
}

func (h *Service) Handshake(
	ctx context.Context,
	peerID core.PeerID,
	stream p2p.Stream,
) (p2p.Peer, error) {

	sig, err := h.createSignature()
	if err != nil {
		return p2p.Peer{}, err
	}

	req := &HandshakeReq{
		PeerType: h.peerType.String(),
		Token:    h.passcode,
		Sig:      sig,
	}

	r, w := msgpack.NewReaderWriter[HandshakeResp, HandshakeReq](stream)

	if err := w.WriteMsg(ctx, req); err != nil {
		return p2p.Peer{}, err
	}

	resp, err := r.ReadMsg(ctx)
	if err != nil {
		return p2p.Peer{}, err
	}

	if !bytes.Equal(resp.ObservedAddress.Bytes(), h.ethAddress.Bytes()) {
		return p2p.Peer{}, errors.New("observed address mismatch")
	}

	if resp.PeerType != h.peerType.String() {
		return p2p.Peer{}, errors.New("peer type mismatch")
	}

	if resp.Ack == nil {
		return p2p.Peer{}, errors.New("ack not received")
	}

	ethAddress, err := h.verifySignature(resp.Ack, peerID)
	if err != nil {
		return p2p.Peer{}, err
	}

	return p2p.Peer{
		EthAddress: ethAddress,
		Type:       p2p.FromString(resp.Ack.PeerType),
	}, nil
}
