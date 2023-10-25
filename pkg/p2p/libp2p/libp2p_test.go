package libp2p_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p"
	registermock "github.com/primevprotocol/mev-commit/pkg/register/mock"
)

func newTestLogger(t *testing.T, w io.Writer) *slog.Logger {
	t.Helper()

	testLogger := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(testLogger)
}

func newTestService(t *testing.T) *libp2p.Service {
	t.Helper()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	svc, err := libp2p.New(&libp2p.Options{
		PrivKey:      privKey,
		Secret:       "test",
		ListenPort:   0,
		PeerType:     p2p.PeerTypeProvider,
		Register:     registermock.New(10),
		MinimumStake: big.NewInt(5),
		Logger:       newTestLogger(t, os.Stdout),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestP2PService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping protocols test in short mode")
	}

	t.Run("new and close", func(t *testing.T) {
		svc := newTestService(t)

		err := svc.Close()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("add protocol and connect", func(t *testing.T) {
		svc := newTestService(t)
		client := newTestService(t)

		t.Cleanup(func() {
			err := errors.Join(svc.Close(), client.Close())
			if err != nil {
				t.Fatal(err)
			}
		})

		done := make(chan struct{})
		svc.AddProtocol(p2p.ProtocolSpec{
			Name:    "test",
			Version: "1.0.0",
			StreamSpecs: []p2p.StreamSpec{
				{
					Name: "test",
					Handler: func(ctx context.Context, peer p2p.Peer, str p2p.Stream) error {
						if peer.EthAddress.Hex() != client.Peer().EthAddress.Hex() {
							t.Fatalf(
								"expected eth address %s, got %s",
								client.Peer().EthAddress.Hex(), peer.EthAddress.Hex(),
							)
						}

						if peer.Type != client.Peer().Type {
							t.Fatalf(
								"expected peer type %s, got %s",
								client.Peer().Type, peer.Type,
							)
						}

						buf, err := str.ReadMsg()
						if err != nil {
							t.Fatal(err)
						}
						if string(buf) != "test" {
							t.Fatalf("expected message %s, got %s", "test", string(buf))
						}
						close(done)
						return nil
					},
				},
			},
		})

		svAddr, err := svc.Addrs()
		if err != nil {
			t.Fatal(err)
		}

		p, err := client.Connect(context.Background(), svAddr)
		if err != nil {
			t.Fatal(err)
		}

		if p.EthAddress.Hex() != svc.Peer().EthAddress.Hex() {
			t.Fatalf(
				"expected eth address %s, got %s",
				svc.Peer().EthAddress.Hex(), p.EthAddress.Hex(),
			)
		}

		if p.Type != svc.Peer().Type {
			t.Fatalf(
				"expected peer type %s, got %s",
				svc.Peer().Type.String(), p.Type.String(),
			)
		}

		str, err := client.NewStream(context.Background(), p, "test", "1.0.0", "test")
		if err != nil {
			t.Fatal(err)
		}

		err = str.WriteMsg([]byte("test"))
		if err != nil {
			t.Fatal(err)
		}

		<-done

		err = str.Close()
		if err != nil {
			t.Fatal(err)
		}

		svcInfo, err := client.GetPeerInfo(svc.Peer())
		if err != nil {
			t.Fatal(err)
		}

		var svcAddr peer.AddrInfo
		err = svcAddr.UnmarshalJSON(svcInfo)
		if err != nil {
			t.Fatal(err)
		}

		if svcAddr.ID != svc.HostID() {
			t.Fatalf("expected host id %s, got %s", svc.HostID(), svcAddr.ID)
		}

		clientInfo, err := svc.GetPeerInfo(client.Peer())
		if err != nil {
			t.Fatal(err)
		}

		var clientAddr peer.AddrInfo
		err = clientAddr.UnmarshalJSON(clientInfo)
		if err != nil {
			t.Fatal(err)
		}

		if clientAddr.ID != client.HostID() {
			t.Fatalf("expected host id %s, got %s", client.HostID(), clientAddr.ID)
		}
	})
}

type testNotifier struct {
	mu    sync.Mutex
	peers []p2p.Peer
}

func (t *testNotifier) Connected(p p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers = append(t.peers, p)
}

func (t *testNotifier) Disconnected(p p2p.Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, peer := range t.peers {
		if peer.EthAddress == p.EthAddress {
			t.peers = append(t.peers[:i], t.peers[i+1:]...)
			return
		}
	}
}

func TestBootstrap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bootstrap test in short mode")
	}

	testDefaultOptions := libp2p.Options{
		Secret:       "test",
		ListenPort:   0,
		PeerType:     p2p.PeerTypeProvider,
		Register:     registermock.New(10),
		MinimumStake: big.NewInt(5),
		Logger:       newTestLogger(t, os.Stdout),
	}

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	bnOpts := testDefaultOptions
	bnOpts.PrivKey = privKey
	bnOpts.PeerType = p2p.PeerTypeBootnode

	bootnode, err := libp2p.New(&bnOpts)
	if err != nil {
		t.Fatal(err)
	}

	notifier := &testNotifier{}
	bootnode.SetNotifier(notifier)

	privKey, err = crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	n1Opts := testDefaultOptions
	n1Opts.BootstrapAddrs = []string{bootnode.AddrString()}
	n1Opts.PrivKey = privKey

	p1, err := libp2p.New(&n1Opts)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	for {
		if time.Since(start) > 10*time.Second {
			t.Fatal("timed out waiting for peers to connect")
		}

		if p1.PeerCount() == 1 {
			if len(notifier.peers) != 1 {
				t.Fatalf("expected 1 peer, got %d", len(notifier.peers))
			}
			if notifier.peers[0].Type != p2p.PeerTypeProvider {
				t.Fatalf(
					"expected peer type %s, got %s",
					p2p.PeerTypeProvider, notifier.peers[0].Type,
				)
			}
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}
