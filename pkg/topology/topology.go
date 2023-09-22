package topology

import "github.com/primevprotocol/mev-commit/pkg/p2p"

type Query struct {
	Type p2p.PeerType
}

type Topology interface {
	p2p.Notifier
	GetPeers(Query) []p2p.Peer
	AddPeers(...p2p.Peer)
}
