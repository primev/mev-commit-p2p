package libp2p

import (
	"context"
	"log/slog"

	core "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p/internal/handshake"
)

type Service struct {
	baseCtx  context.Context
	host     host.Host
	peers    *peerRegistry
	logger   *slog.Logger
	notifier p2p.Notifier
	hsSvc    *handshake.Service
}

func New(baseCtx context.Context, host host.Host, logger *slog.Logger) *Service {
	s := &Service{
		baseCtx: baseCtx,
		host:    host,
		peers:   newPeerRegistry(),
		logger:  logger,
	}
	s.peers.setDisconnector(s)

	s.host.SetStreamHandler(handshake.ProtocolID(), s.handleConnectReq)
	return s
}

func (s *Service) handleConnectReq(streamlibp2p network.Stream) {
	peerID := streamlibp2p.Conn().RemotePeer()

	stream := newStream(streamlibp2p)
	peer, err := s.hsSvc.Handle(s.baseCtx, stream, peerID)
	if err != nil {
		s.logger.Error("error handling handshake", "err", err)
		_ = streamlibp2p.Reset()
		return
	}

	if exists := s.peers.addPeer(streamlibp2p.Conn(), peer); exists {
		s.logger.Warn("peer already exists", "peer", peer)
		_ = streamlibp2p.Reset()
		return
	}

	if s.notifier != nil {
		s.notifier.Connected(peer)
	}

	s.logger.Info("peer connected (inbound)", "peer", peer)
}

func (s *Service) disconnected(p p2p.Peer) {
	s.notifier.Disconnected(p)
}

func (s *Service) AddProtocol(spec p2p.ProtocolSpec) {
	for _, streamSpec := range spec.StreamSpecs {
		ss := streamSpec
		id := protocol.ID(p2p.NewStreamName(spec.Name, spec.Version, ss.Name))

		// TODO: If we need semantic versioning, we need to use the
		// SetStreamHandlerMatch function instead of SetStreamHandler
		s.host.SetStreamHandler(id, func(streamlibp2p network.Stream) {
			peerID := streamlibp2p.Conn().RemotePeer()
			p, found := s.peers.getPeer(peerID)
			if !found {
				s.logger.Error("received stream from unknown peer", "peer", peerID)
				_ = streamlibp2p.Reset()
				return
			}

			ctx, cancel := context.WithCancel(s.baseCtx)
			s.peers.addStream(peerID, streamlibp2p, cancel)
			defer s.peers.removeStream(peerID, streamlibp2p)

			stream := newStream(streamlibp2p)

			if err := ss.Handler(ctx, p, stream); err != nil {
				s.logger.Error("error handling stream", "err", err)
			}
		})
	}
}

func (s *Service) NewStream(
	ctx context.Context,
	peer p2p.Peer,
	proto string,
	version string,
	streamName string,
) (p2p.Stream, error) {

	peerID, found := s.peers.getPeerID(peer.EthAddress)
	if !found {
		return nil, p2p.ErrPeerNotFound
	}

	streamID := protocol.ID(p2p.NewStreamName(proto, version, streamName))
	streamlibp2p, err := s.host.NewStream(ctx, peerID, streamID)
	if err != nil {
		return nil, err
	}

	return newStream(streamlibp2p), nil
}

func (s *Service) newStreamForPeer(
	ctx context.Context,
	peerID core.PeerID,
	streamID protocol.ID,
) (p2p.Stream, error) {
	streamlibp2p, err := s.host.NewStream(ctx, peerID, streamID)
	if err != nil {
		return nil, err
	}

	return newStream(streamlibp2p), nil
}

func (s *Service) Connect(ctx context.Context, info []byte) (p2p.Peer, error) {
	var addrInfo peer.AddrInfo
	if err := addrInfo.UnmarshalJSON(info); err != nil {
		return p2p.Peer{}, err
	}

	if len(addrInfo.Addrs) == 0 {
		return p2p.Peer{}, p2p.ErrNoAddresses
	}

	if p, found := s.peers.isConnected(addrInfo.ID); found {
		return p, nil
	}

	if err := s.host.Connect(ctx, addrInfo); err != nil {
		return p2p.Peer{}, err
	}

	stream, err := s.newStreamForPeer(ctx, addrInfo.ID, handshake.ProtocolID())
	if err != nil {
		return p2p.Peer{}, err
	}

	p, err := s.hsSvc.Handshake(ctx, addrInfo.ID, stream)
	if err != nil {
		return p2p.Peer{}, err
	}

	s.host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)

	return p, nil
}

func (s *Service) GetPeerInfo(p p2p.Peer) ([]byte, error) {
	peerID, found := s.peers.getPeerID(p.EthAddress)
	if !found {
		return nil, p2p.ErrPeerNotFound
	}

	peerInfo := s.host.Peerstore().PeerInfo(peerID)
	return peerInfo.MarshalJSON()
}
