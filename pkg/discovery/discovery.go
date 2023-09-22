package discovery

import (
	"context"

	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"golang.org/x/sync/semaphore"
)

const (
	ProtocolName    = "discovery"
	ProtocolVersion = "1.0.0"
)

type P2PService interface {
	p2p.Streamer
	p2p.Addressbook
	Connect(context.Context, []byte) (p2p.Peer, error)
}

type Topology interface {
	AddPeers(...p2p.Peer)
}

type Discovery struct {
	topo       Topology
	streamer   P2PService
	checkPeers chan PeerInfo
	sem        *semaphore.Weighted
	quit       chan struct{}
}

func New(topo Topology, streamer P2PService) *Discovery {
	return &Discovery{
		topo:     topo,
		streamer: streamer,
	}
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

type PeerInfo struct {
	ID       string
	PeerType string
	Underlay []byte
}

type peersList struct {
	Peers []PeerInfo
}

func (d *Discovery) handlePeersList(ctx context.Context, peer p2p.Peer, s p2p.Stream) error {
	r, _ := msgpack.NewReaderWriter[peersList, peersList](s)

	peers, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	for _, p := range peers.Peers {
		select {
		case d.checkPeers <- p:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (d *Discovery) BroadcastPeers(ctx context.Context, peer p2p.Peer, peers []PeerInfo) error {
	stream, err := d.streamer.NewStream(ctx, peer, ProtocolName, ProtocolVersion, "peersList")
	if err != nil {
		return err
	}

	_, w := msgpack.NewReaderWriter[peersList, peersList](stream)
	if err := w.WriteMsg(ctx, &peersList{Peers: peers}); err != nil {
		return err
	}

	return nil
}

func (d *Discovery) checkAndAddPeers() {
	for {
		select {
		case <-d.quit:
			return
		case peer := <-d.checkPeers:
			d.sem.Acquire(context.Background(), 1)
			go func() {
				defer d.sem.Release(1)

				p, err := d.streamer.Connect(context.Background(), peer.Underlay)
				if err != nil {
					return
				}
				d.topo.AddPeers(p)
			}()
		}
	}
}
