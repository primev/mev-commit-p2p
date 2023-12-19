package libp2p

import (
	"time"

	core "github.com/libp2p/go-libp2p/core"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
)

type blockInfo struct {
	reason string
	until  time.Time
}

func (s *Service) blockPeer(peer core.PeerID, dur time.Duration, reason string) {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	s.blockMap[peer] = blockInfo{
		reason: reason,
		until:  time.Now().Add(dur),
	}
}

func (s *Service) isBlocked(peer core.PeerID) bool {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	info, ok := s.blockMap[peer]
	if !ok {
		return false
	}
	if time.Now().After(info.until) {
		delete(s.blockMap, peer)
		return false
	}
	return true
}

func (s *Service) BlockedPeers() []p2p.BlockedPeerInfo {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	var res []p2p.BlockedPeerInfo
	for id, info := range s.blockMap {
		if time.Now().After(info.until) {
			ethAddr, err := GetEthAddressFromPeerID(id)
			if err != nil {
				continue
			}
			durString := info.until.Sub(time.Now()).String()
			if durString == "0s" {
				durString = "Forever"
			}
			res = append(res, p2p.BlockedPeerInfo{
				Peer:     ethAddr,
				Reason:   info.reason,
				Duration: durString,
			})

		}
	}
	return res
}
