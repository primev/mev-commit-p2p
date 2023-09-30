package preconfirmation_test

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

type testTopo struct {
	mu    sync.Mutex
	peers []p2p.Peer
}

func (t *testTopo) AddPeers(peers ...p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.peers = append(t.peers, peers...)
}

func (t *testTopo) GetPeers(q topology.Query) []p2p.Peer {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.peers
}

func (t *testTopo) Connected(p2p.Peer) {
}

func (t *testTopo) Disconnected(p2p.Peer) {
}

func TestPreconfBidSubmission(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		client := p2p.Peer{
			EthAddress: common.HexToAddress("0x1"),
			Type:       p2p.PeerTypeSearcher,
		}
		server := p2p.Peer{
			EthAddress: common.HexToAddress("0x2"),
			Type:       p2p.PeerTypeBuilder,
		}

		svc := p2ptest.New(
			p2ptest.WithConnectFunc(func(addr []byte) (p2p.Peer, error) {
				if string(addr) != "test" {
					return p2p.Peer{}, errors.New("invalid address")
				}
				return client, nil
			}),
			p2ptest.WithAddressbookFunc(func(p p2p.Peer) ([]byte, error) {
				if p.EthAddress != client.EthAddress {
					return nil, errors.New("invalid peer")
				}
				return []byte("test"), nil
			}),
		)

		topo := &testTopo{}
		key, _ := crypto.GenerateKey()
		p := preconfirmation.New(topo, svc, key)

		svc.SetPeerHandler(server, p.Protocol())
		err := p.SendBid(context.Background(), "0x4c03a845396b770ad41b975d6bd3bf8c2bd5cca36867a3301f9598f2e3e9518d", big.NewInt(10), big.NewInt(10))
		if err != nil {
			t.Fatal(err)
		}

	})
}
