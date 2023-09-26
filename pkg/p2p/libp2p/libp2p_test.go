package libp2p_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"os"
	"testing"

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
		PeerType:     p2p.PeerTypeBuilder,
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
	t.Parallel()

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
