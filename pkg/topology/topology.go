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
	BroadcastPeers(context.Context, p2p.Peer, []p2p.PeerInfo) error
}

type topology struct {
	mu          sync.RWMutex
	providers   map[common.Address]p2p.Peer
	users       map[common.Address]p2p.Peer
	logger      *slog.Logger
	addressbook p2p.Addressbook
	announcer   Announcer
}

func New(a p2p.Addressbook, logger *slog.Logger) *topology {
	return &topology{
		providers:   make(map[common.Address]p2p.Peer),
		users:       make(map[common.Address]p2p.Peer),
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
		// Whether its a provider or user, we want to broadcast the provider peers
		peersToBroadcast := t.GetPeers(Query{Type: p2p.PeerTypeProvider})
		var underlays []p2p.PeerInfo
		for _, peer := range peersToBroadcast {
			if peer.EthAddress == p.EthAddress {
				continue
			}
			u, err := t.addressbook.GetPeerInfo(peer)
			if err != nil {
				t.logger.Error("failed to get peer info", "err", err, "peer", peer)
				continue
			}
			underlays = append(underlays, p2p.PeerInfo{
				EthAddress: peer.EthAddress,
				Underlay:   u,
			})
		}

		if len(underlays) > 0 {
			err := t.announcer.BroadcastPeers(context.Background(), p, underlays)
			if err != nil {
				t.logger.Error("failed to broadcast peers", "err", err, "peer", p)
			}
		}

		if p.Type == p2p.PeerTypeProvider {
			t.logger.Info("provider connected broadcasting to previous users", "peer", p)
			// If the peer is a provider, we want to broadcast to the user peers
			peersToBroadcastTo := t.GetPeers(Query{Type: p2p.PeerTypeUser})
			providerUnderlay, err := t.addressbook.GetPeerInfo(p)
			if err != nil {
				t.logger.Error("failed to get peer info", "err", err, "peer", p)
				return
			}
			for _, peer := range peersToBroadcastTo {
				err := t.announcer.BroadcastPeers(context.Background(), peer, []p2p.PeerInfo{
					{
						EthAddress: p.EthAddress,
						Underlay:   providerUnderlay,
					},
				})
				if err != nil {
					t.logger.Error("failed to broadcast peer", "err", err, "peer", peer)
				}
			}
		}
	}
}

func (t *topology) add(p p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch p.Type {
	case p2p.PeerTypeProvider:
		t.providers[p.EthAddress] = p
	case p2p.PeerTypeUser:
		t.users[p.EthAddress] = p
	}
}

func (t *topology) Disconnected(p p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Info("disconnected", "peer", p)

	switch p.Type {
	case p2p.PeerTypeProvider:
		delete(t.providers, p.EthAddress)
	case p2p.PeerTypeUser:
		delete(t.users, p.EthAddress)
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
	case p2p.PeerTypeProvider:
		for _, p := range t.providers {
			peers = append(peers, p)
		}
	case p2p.PeerTypeUser:
		for _, p := range t.users {
			peers = append(peers, p)
		}
	}

	return peers
}

func (t *topology) IsConnected(addr common.Address) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, ok := t.providers[addr]; ok {
		return true
	}

	if _, ok := t.users[addr]; ok {
		return true
	}

	return false
}
