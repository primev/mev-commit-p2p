package preconfirmation_test

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/structures/preconf"
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

	return []p2p.Peer{{EthAddress: common.HexToAddress("0x2"), Type: p2p.PeerTypeSearcher}}
}

func (t *testTopo) Connected(p2p.Peer) {
}

func (t *testTopo) Disconnected(p2p.Peer) {
}

type testCommitmentStore struct {
}

func (t *testCommitmentStore) GetCommitments(bidHash []byte) ([]preconf.PreconfCommitment, error) {
	return []preconf.PreconfCommitment{}, nil
}

func (t *testCommitmentStore) AddCommitment(bidHash []byte, commitment *preconf.PreconfCommitment) error {
	return nil
}

type testUserStore struct {
}

func (t *testUserStore) CheckUserRegistred(_ *common.Address) bool {
	return true
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
			Type:       p2p.PeerTypeSearcher,
		}

		svc := p2ptest.New(
			p2ptest.WithConnectFunc(func(addr []byte) (p2p.Peer, error) {
				return client, nil
			}),
		)

		topo := &testTopo{}
		us := &testUserStore{}
		cs := &testCommitmentStore{}
		key, _ := crypto.GenerateKey()
		p := preconfirmation.New(topo, svc, key, us, cs)

		// svc.SetPeerHandler(client, p.Protocol())
		svc.SetPeerHandler(server, p.Protocol())

		err := p.SendBid(context.Background(), "0x4c03a845396b770ad41b975d6bd3bf8c2bd5cca36867a3301f9598f2e3e9518d", big.NewInt(10), big.NewInt(10))
		if err != nil {
			t.Fatal(err)
		}

	})
}
