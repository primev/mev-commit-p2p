package discovery

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"golang.org/x/sync/semaphore"
)

const (
	ProtocolName    = "discovery"
	ProtocolVersion = "1.0.0"
	checkWorkers    = 10
)

type P2PService interface {
	p2p.Streamer
	Connect(context.Context, []byte) (p2p.Peer, error)
}

type Topology interface {
	AddPeers(...p2p.Peer)
	IsConnected(common.Address) bool
}

type Discovery struct {
	topo       Topology
	streamer   P2PService
	logger     *slog.Logger
	checkPeers chan p2p.PeerInfo
	sem        *semaphore.Weighted
	quit       chan struct{}
}

func New(
	topo Topology,
	streamer P2PService,
	logger *slog.Logger,
) *Discovery {
	d := &Discovery{
		topo:       topo,
		streamer:   streamer,
		logger:     logger.With("protocol", ProtocolName),
		sem:        semaphore.NewWeighted(checkWorkers),
		checkPeers: make(chan p2p.PeerInfo),
		quit:       make(chan struct{}),
	}
	go d.checkAndAddPeers()
	return d
}

func (d *Discovery) Protocol() p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{
				Name:    "peersList",
				Handler: d.handlePeersList,
			},
		},
	}
}

type peersList struct {
	Peers []p2p.PeerInfo
}

func (d *Discovery) handlePeersList(ctx context.Context, peer p2p.Peer, s p2p.Stream) error {
	r, _ := msgpack.NewReaderWriter[peersList, peersList](s)

	peers, err := r.ReadMsg(ctx)
	if err != nil {
		d.logger.Error("failed to read peers list", "err", err, "from_peer", peer)
		return err
	}

	for _, p := range peers.Peers {
		if d.topo.IsConnected(p.EthAddress) {
			continue
		}
		select {
		case d.checkPeers <- p:
		case <-ctx.Done():
			d.logger.Error("failed to add peer", "err", ctx.Err(), "from_peer", peer)
			return ctx.Err()
		}
	}

	d.logger.Debug("added peers", "peers", len(peers.Peers), "from_peer", peer)
	return nil
}

func (d *Discovery) BroadcastPeers(
	ctx context.Context,
	peer p2p.Peer,
	peers []p2p.PeerInfo,
) error {
	stream, err := d.streamer.NewStream(ctx, peer, ProtocolName, ProtocolVersion, "peersList")
	if err != nil {
		d.logger.Error("failed to create stream", "err", err, "to_peer", peer)
		return err
	}
	defer stream.Close()

	_, w := msgpack.NewReaderWriter[peersList, peersList](stream)
	if err := w.WriteMsg(ctx, &peersList{Peers: peers}); err != nil {
		d.logger.Error("failed to write peers list", "err", err, "to_peer", peer)
		return err
	}

	d.logger.Debug("sent peers list", "peers", len(peers), "to_peer", peer)
	return nil
}

func (d *Discovery) Close() error {
	close(d.quit)
	return nil
}

func (d *Discovery) checkAndAddPeers() {
	for {
		select {
		case <-d.quit:
			return
		case peer := <-d.checkPeers:
			_ = d.sem.Acquire(context.Background(), 1)
			go func() {
				defer d.sem.Release(1)

				p, err := d.streamer.Connect(context.Background(), peer.Underlay)
				if err != nil {
					d.logger.Error("failed to connect to peer", "err", err, "peer", peer)
					return
				}
				d.topo.AddPeers(p)
			}()
		}
	}
}
