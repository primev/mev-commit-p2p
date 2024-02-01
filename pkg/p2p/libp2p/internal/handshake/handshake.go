package handshake

import (
	"bytes"
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/primevprotocol/mev-commit/pkg/keysigner"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/signer"
)

const (
	ProtocolName    = "handshake"
	ProtocolVersion = "1.0.0"
	StreamName      = "handshake"
)

var (
	ErrSignatureVerificationFailed = errors.New("signature verification failed")
	ErrObservedAddressMismatch     = errors.New("observed address mismatch")
	ErrInsufficientStake           = errors.New("insufficient stake")
)

type ProviderRegistry interface {
	CheckProviderRegistered(context.Context, common.Address) bool
}

// Handshake is the handshake protocol
type Service struct {
	ks            keysigner.KeySigner
	peerType      p2p.PeerType
	passcode      string
	signer        signer.Signer
	register      ProviderRegistry
	handshakeReq  *HandshakeReq
	getEthAddress func(core.PeerID) (common.Address, error)
}

func New(
	ks keysigner.KeySigner,
	peerType p2p.PeerType,
	passcode string,
	signer signer.Signer,
	register ProviderRegistry,
	getEthAddress func(core.PeerID) (common.Address, error),
) (*Service, error) {
	s := &Service{
		ks:            ks,
		peerType:      peerType,
		passcode:      passcode,
		signer:        signer,
		register:      register,
		getEthAddress: getEthAddress,
	}

	err := s.setHandshakeReq()
	if err != nil {
		return nil, err
	}

	return s, nil
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
}

func (h *Service) verifyReq(
	req *HandshakeReq,
	peerID core.PeerID,
) (common.Address, error) {
	unsignedData := []byte(req.PeerType + req.Token)

	verified, ethAddress, err := h.signer.Verify(req.Sig, unsignedData)
	if err != nil {
		return common.Address{}, errors.Join(err, ErrSignatureVerificationFailed)
	}

	if !verified {
		return common.Address{}, ErrSignatureVerificationFailed
	}

	observedEthAddress, err := h.getEthAddress(peerID)
	if err != nil {
		return common.Address{}, err
	}

	if !bytes.Equal(observedEthAddress.Bytes(), ethAddress.Bytes()) {
		return common.Address{}, ErrObservedAddressMismatch
	}

	if req.PeerType == p2p.PeerTypeProvider.String() {
		if !h.register.CheckProviderRegistered(context.Background(), ethAddress) {
			return common.Address{}, ErrInsufficientStake
		}
	}

	return ethAddress, nil
}

func (h *Service) createSignature() ([]byte, error) {
	unsignedData := []byte(h.peerType.String() + h.passcode)
	hash := crypto.Keccak256Hash(unsignedData)
	sig, err := h.ks.SignHash(hash.Bytes())
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func (h *Service) setHandshakeReq() error {
	sig, err := h.createSignature()
	if err != nil {
		return err
	}

	req := &HandshakeReq{
		PeerType: h.peerType.String(),
		Token:    h.passcode,
		Sig:      sig,
	}

	h.handshakeReq = req
	return nil
}

func (h *Service) verifyResp(resp *HandshakeResp) error {
	if !bytes.Equal(resp.ObservedAddress.Bytes(), h.ks.GetAddress().Bytes()) {
		return errors.New("observed address mismatch")
	}

	if resp.PeerType != h.peerType.String() {
		return errors.New("peer type mismatch")
	}

	return nil
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

	ethAddress, err := h.verifyReq(req, peerID)
	if err != nil {
		return p2p.Peer{}, err
	}

	resp := &HandshakeResp{
		ObservedAddress: ethAddress,
		PeerType:        req.PeerType,
	}

	if err := w.WriteMsg(ctx, resp); err != nil {
		return p2p.Peer{}, err
	}

	ar, aw := msgpack.NewReaderWriter[HandshakeResp, HandshakeReq](stream)

	err = aw.WriteMsg(ctx, h.handshakeReq)
	if err != nil {
		return p2p.Peer{}, err
	}

	ack, err := ar.ReadMsg(ctx)
	if err != nil {
		return p2p.Peer{}, err
	}

	if err := h.verifyResp(ack); err != nil {
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

	r, w := msgpack.NewReaderWriter[HandshakeResp, HandshakeReq](stream)

	if err := w.WriteMsg(ctx, h.handshakeReq); err != nil {
		return p2p.Peer{}, err
	}

	resp, err := r.ReadMsg(ctx)
	if err != nil {
		return p2p.Peer{}, err
	}

	if err := h.verifyResp(resp); err != nil {
		return p2p.Peer{}, err
	}

	ar, aw := msgpack.NewReaderWriter[HandshakeReq, HandshakeResp](stream)

	ack, err := ar.ReadMsg(ctx)
	if err != nil {
		return p2p.Peer{}, err
	}

	ethAddress, err := h.verifyReq(ack, peerID)
	if err != nil {
		return p2p.Peer{}, err
	}

	err = aw.WriteMsg(ctx, &HandshakeResp{
		ObservedAddress: ethAddress,
		PeerType:        ack.PeerType,
	})
	if err != nil {
		return p2p.Peer{}, err
	}

	return p2p.Peer{
		EthAddress: ethAddress,
		Type:       p2p.FromString(ack.PeerType),
	}, nil
}
