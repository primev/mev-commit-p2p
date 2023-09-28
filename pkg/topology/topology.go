package topology

import (
	"context"
	"log/slog"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
)

type Query struct {
	Type p2p.PeerType
}

type Topology interface {
	p2p.Notifier
	SetAnnouncer(Announcer)
	GetPeers(Query) []p2p.Peer
	AddPeers(...p2p.Peer)
	IsConnected(common.Address) bool
}

type Announcer interface {
	BroadcastPeers(context.Context, p2p.Peer, [][]byte) error
}

type topology struct {
	mu          sync.RWMutex
	builders    map[common.Address]p2p.Peer
	searchers   map[common.Address]p2p.Peer
	logger      *slog.Logger
	addressbook p2p.Addressbook
	announcer   Announcer
}

func New(a p2p.Addressbook, logger *slog.Logger) Topology {
	return &topology{
		builders:    make(map[common.Address]p2p.Peer),
		searchers:   make(map[common.Address]p2p.Peer),
		addressbook: a,
		logger:      logger,
	}
}

func (t *topology) SetAnnouncer(a Announcer) {
	t.announcer = a
}

func (t *topology) Connected(p p2p.Peer) {
	t.add(p)

	if t.announcer != nil {
		// Whether its a builder or searcher, we want to broadcast the builder peers
		peersToBroadcast := t.GetPeers(Query{Type: p2p.PeerTypeBuilder})
		var underlays [][]byte
		for _, peer := range peersToBroadcast {
			if peer.EthAddress == p.EthAddress {
				continue
			}
			u, err := t.addressbook.GetPeerInfo(peer)
			if err != nil {
				t.logger.Error("failed to get peer info", "err", err, "peer", peer)
				continue
			}
			underlays = append(underlays, u)
		}

		if len(underlays) == 0 {
			t.logger.Warn("no underlays to broadcast", "peer", p)
			return
		}
		err := t.announcer.BroadcastPeers(context.Background(), p, underlays)
		if err != nil {
			t.logger.Error("failed to broadcast peers", "err", err, "peer", p)
		}
	}
}

func (t *topology) add(p p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch p.Type {
	case p2p.PeerTypeBuilder:
		t.builders[p.EthAddress] = p
	case p2p.PeerTypeSearcher:
		t.searchers[p.EthAddress] = p
	}
}

func (t *topology) Disconnected(p p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch p.Type {
	case p2p.PeerTypeBuilder:
		delete(t.builders, p.EthAddress)
	case p2p.PeerTypeSearcher:
		delete(t.searchers, p.EthAddress)
	}
}

func (t *topology) AddPeers(peers ...p2p.Peer) {
	for _, p := range peers {
		t.add(p)
	}
}

func (t *topology) GetPeers(q Query) []p2p.Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var peers []p2p.Peer

	switch q.Type {
	case p2p.PeerTypeBuilder:
		for _, p := range t.builders {
			peers = append(peers, p)
		}
	case p2p.PeerTypeSearcher:
		for _, p := range t.searchers {
			peers = append(peers, p)
		}
	}

	return peers
}

func (t *topology) IsConnected(addr common.Address) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, ok := t.builders[addr]; ok {
		return true
	}

	if _, ok := t.searchers[addr]; ok {
		return true
	}

	return false
}
