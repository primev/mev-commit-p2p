package libp2p

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func (p *Service) startBootstrapper(addrs []string) {
	for {
		for _, addr := range addrs {
			addrInfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				p.logger.Error("failed to parse bootstrap address", "addr", addr, "err", err)
				continue
			}

			if _, connected := p.peers.isConnected(addrInfo.ID); connected {
				p.logger.Debug("already connected to bootstrap peer", "peer", addrInfo.ID)
				continue
			}

			addrInfoBytes, err := addrInfo.MarshalJSON()
			if err != nil {
				p.logger.Error("failed to marshal bootstrap peer", "addr", addr, "err", err)
				continue
			}

			peer, err := p.Connect(context.Background(), addrInfoBytes)
			if err != nil {
				p.logger.Error("failed to connect to bootstrap peer", "addr", addr, "err", err)
				continue
			}

			p.logger.Info("connected to bootstrap peer", "peer", peer)
		}
		time.Sleep(1 * time.Minute)
	}
}
