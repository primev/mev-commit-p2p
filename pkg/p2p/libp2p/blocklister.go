package libp2p

import (
	"time"

	core "github.com/libp2p/go-libp2p/core"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
)

type blockInfo struct {
	reason   string
	start    time.Time
	duration time.Duration
}

func (s *Service) blockPeer(peer core.PeerID, dur time.Duration, reason string) {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	s.blockMap[peer] = blockInfo{
		reason:   reason,
		start:    time.Now(),
		duration: dur,
	}
}

func (s *Service) isBlocked(peer core.PeerID) bool {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	info, ok := s.blockMap[peer]
	if !ok {
		return false
	}
	if time.Now().After(info.start.Add(info.duration)) && info.duration != 0 {
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
		if time.Now().Before(info.start.Add(info.duration)) || info.duration == 0 {
			ethAddr, err := GetEthAddressFromPeerID(id)
			if err != nil {
				continue
			}
			var durString string
			if info.duration == 0 {
				durString = "Forever"
			} else {
				durString = info.start.Add(info.duration).Sub(time.Now()).String()
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
