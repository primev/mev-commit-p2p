package node

import (
	"crypto/ecdsa"
	"log/slog"

	"github.com/primevprotocol/mev-commit/pkg/discovery"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	"github.com/primevprotocol/mev-commit/pkg/register"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"github.com/prometheus/client_golang/prometheus"
)

type Options struct {
	PrivKey    *ecdsa.PrivateKey
	Secret     string
	PeerType   string
	Logger     *slog.Logger
	MetricsReg *prometheus.Registry
	ListenPort int
}

type Node struct {
}

func NewNode(opts *Options) (*Node, error) {
	reg := register.New()

	minStake, err := reg.GetMinimumStake()
	if err != nil {
		return nil, err
	}

	peerType := p2p.FromString(opts.PeerType)

	p2pSvc, err := libp2p.New(&libp2p.Options{
		PrivKey:      opts.PrivKey,
		Secret:       opts.Secret,
		PeerType:     peerType,
		Register:     reg,
		MinimumStake: minStake,
		Logger:       opts.Logger,
		MetricsReg:   opts.MetricsReg,
		ListenPort:   opts.ListenPort,
	})

	topo := topology.New(p2pSvc, opts.Logger)
	disc := discovery.New(topo, p2pSvc, opts.Logger)

	// Set the announcer for the topology service
	topo.SetAnnouncer(disc)
	// Set the notifier for the p2p service
	p2pSvc.SetNotifier(topo)

	// Register the discovery protocol with the p2p service
	p2pSvc.AddProtocol(disc.Protocol())

	return &Node{}, nil
}
