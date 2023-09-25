package discovery_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/discovery"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
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

func newTestLogger(w io.Writer) *slog.Logger {
	testLogger := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(testLogger)
}

func TestDiscovery(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		client := p2p.Peer{
			EthAddress: common.HexToAddress("0x1"),
			Type:       p2p.PeerTypeBuilder,
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
		d := discovery.New(topo, svc, newTestLogger(os.Stdout))
		t.Cleanup(func() {
			err := d.Close()
			if err != nil {
				t.Fatal(err)
			}
		})

		svc.SetPeerHandler(server, d.Protocol())

		err := d.BroadcastPeers(context.Background(), server, []discovery.PeerInfo{
			{
				ID:       common.HexToAddress("0x1").Hex(),
				PeerType: p2p.PeerTypeBuilder.String(),
				Underlay: []byte("test"),
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		start := time.Now()
		for {
			if time.Since(start) > 5*time.Second {
				t.Fatal("timed out")
			}
			if len(topo.peers) == 1 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}
