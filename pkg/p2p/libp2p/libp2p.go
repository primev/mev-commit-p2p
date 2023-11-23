package libp2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/primevprotocol/mev-commit/pkg/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p/internal/handshake"
	"github.com/primevprotocol/mev-commit/pkg/register"
	"github.com/primevprotocol/mev-commit/pkg/signer"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultMetricsNamespace = "mev_commit"
)

type Service struct {
	baseCtx       context.Context
	baseCtxCancel context.CancelFunc
	ethAddress    common.Address
	peerType      p2p.PeerType
	host          host.Host
	peers         *peerRegistry
	logger        *slog.Logger
	notifier      p2p.Notifier
	hsSvc         *handshake.Service
	metrics       *metrics
}

type Options struct {
	PrivKey        *ecdsa.PrivateKey
	Secret         string
	PeerType       p2p.PeerType
	Register       register.Register
	MinimumStake   *big.Int
	ListenPort     int
	Logger         *slog.Logger
	MetricsReg     *prometheus.Registry
	BootstrapAddrs []string
}

func New(opts *Options) (*Service, error) {
	padded32BytePrivKey := util.PadKeyTo32Bytes(opts.PrivKey.D)
	libp2pKey, err := libp2pcrypto.UnmarshalSecp256k1PrivateKey(padded32BytePrivKey)
	if err != nil {
		return nil, err
	}

	connmgr, err := connmgr.NewConnManager(
		100, // Lowwater
		400, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, err
	}

	pstore, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, err
	}

	var metrics = new(metrics)
	if opts.MetricsReg != nil {
		rcmgr.MustRegisterWith(opts.MetricsReg)
		metrics = newMetrics(opts.MetricsReg, defaultMetricsNamespace)
	}

	str, err := rcmgr.NewStatsTraceReporter()
	if err != nil {
		return nil, err
	}

	cfg := rcmgr.NewFixedLimiter(rcmgr.DefaultLimits.AutoScale())

	rmgr, err := rcmgr.NewResourceManager(cfg, rcmgr.WithTraceReporter(str))
	if err != nil {
		return nil, err
	}

	conngtr := newConnectionGater(
		opts.Register,
		opts.PeerType,
		opts.MinimumStake,
		metrics,
	)

	host, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", opts.ListenPort)),
		libp2p.ConnectionGater(conngtr),
		libp2p.Identity(libp2pKey),
		libp2p.ConnectionManager(connmgr),
		libp2p.DefaultTransports,
		libp2p.DefaultSecurity,
		libp2p.Peerstore(pstore),
		libp2p.ResourceManager(rmgr),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
	)
	if err != nil {
		return nil, err
	}

	for _, addr := range host.Addrs() {
		opts.Logger.Info("p2p address", "addr", addr, "host_address", host.ID().Pretty())
	}

	ethAddress, err := GetEthAddressFromPeerID(host.ID())
	if err != nil {
		return nil, err
	}

	hsSvc, err := handshake.New(
		opts.PrivKey,
		ethAddress,
		opts.PeerType,
		opts.Secret,
		signer.New(),
		opts.Register,
		opts.MinimumStake,
		GetEthAddressFromPeerID,
	)
	if err != nil {
		return nil, err
	}

	baseCtx, baseCtxCancel := context.WithCancel(context.Background())

	s := &Service{
		baseCtx:       baseCtx,
		baseCtxCancel: baseCtxCancel,
		ethAddress:    ethAddress,
		peerType:      opts.PeerType,
		host:          host,
		peers:         newPeerRegistry(),
		hsSvc:         hsSvc,
		logger:        opts.Logger,
		metrics:       metrics,
	}
	s.peers.setDisconnector(s)

	host.Network().Notify(s.peers)

	s.host.SetStreamHandler(handshake.ProtocolID(), s.handleConnectReq)

	if len(opts.BootstrapAddrs) > 0 {
		go s.startBootstrapper(opts.BootstrapAddrs)
	}
	return s, nil
}

func (s *Service) Close() error {
	s.baseCtxCancel()
	return s.host.Close()
}

func (s *Service) SetNotifier(n p2p.Notifier) {
	s.notifier = n
}

func (s *Service) handleConnectReq(streamlibp2p network.Stream) {
	peerID := streamlibp2p.Conn().RemotePeer()

	stream := newStream(streamlibp2p)
	peer, err := s.hsSvc.Handle(s.baseCtx, stream, peerID)
	if err != nil {
		s.logger.Error("error handling handshake", "err", err)
		_ = streamlibp2p.Reset()
		s.metrics.FailedIncomingHandshakeCount.Inc()
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
	if s.notifier != nil {
		s.notifier.Disconnected(p)

	}
}

func (s *Service) Self() map[string]interface{} {
	return map[string]interface{}{
		"Ethereum Address": s.ethAddress.Hex(),
		"Peer Type":        s.peerType.String(),
		"Underlay":         s.host.ID().String(),
		"Addresses":        s.host.Addrs(),
	}
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

	streamlibp2p, err := s.host.NewStream(ctx, addrInfo.ID, handshake.ProtocolID())
	if err != nil {
		return p2p.Peer{}, err
	}
	stream := newStream(streamlibp2p)

	p, err := s.hsSvc.Handshake(ctx, addrInfo.ID, stream)
	if err != nil {
		_ = s.host.Network().ClosePeer(addrInfo.ID)
		s.metrics.FailedOutgoingHandshakeCount.Inc()
		return p2p.Peer{}, err
	}

	if exists := s.peers.addPeer(streamlibp2p.Conn(), p); exists {
		s.logger.Warn("peer already exists", "peer", p)
	}

	s.host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)
	s.logger.Info("peer connected (outbound)", "peer", p)

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
