package handshake_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p/internal/handshake"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
)

type testRegister struct{}

func (t *testRegister) CheckProviderRegistered(
	_ context.Context,
	_ common.Address,
) bool {
	return true
}

type testSigner struct {
	address common.Address
}

func (t *testSigner) Sign(_ *ecdsa.PrivateKey, _ []byte) ([]byte, error) {
	return []byte("signature"), nil
}

func (t *testSigner) Verify(_ []byte, _ []byte) (bool, common.Address, error) {
	return true, t.address, nil
}

func TestHandshake(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		privKey1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}

		privKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}

		hs1, err := handshake.New(
			privKey1,
			common.HexToAddress("0x1"),
			p2p.PeerTypeProvider,
			"test",
			&testSigner{address: common.HexToAddress("0x2")},
			&testRegister{},
			func(p core.PeerID) (common.Address, error) {
				return common.HexToAddress("0x2"), nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		hs2, err := handshake.New(
			privKey2,
			common.HexToAddress("0x2"),
			p2p.PeerTypeProvider,
			"test",
			&testSigner{address: common.HexToAddress("0x1")},
			&testRegister{},
			func(p core.PeerID) (common.Address, error) {
				return common.HexToAddress("0x1"), nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		out, in := p2ptest.NewDuplexStream()

		done := make(chan struct{})
		go func() {
			defer close(done)

			p, err := hs1.Handle(context.Background(), in, core.PeerID("test2"))
			if err != nil {
				t.Error(err)
				return
			}
			if p.EthAddress != common.HexToAddress("0x2") {
				t.Errorf(
					"expected eth address %s, got %s",
					common.HexToAddress("0x2"), p.EthAddress,
				)
				return
			}
			if p.Type != p2p.PeerTypeProvider {
				t.Errorf("expected peer type %s, got %s", p2p.PeerTypeProvider, p.Type)
				return
			}
		}()

		p, err := hs2.Handshake(context.Background(), core.PeerID("test1"), out)
		if err != nil {
			t.Fatal(err)
		}
		if p.EthAddress != common.HexToAddress("0x1") {
			t.Fatalf("expected eth address %s, got %s", common.HexToAddress("0x1"), p.EthAddress)
		}
		if p.Type != p2p.PeerTypeProvider {
			t.Fatalf("expected peer type %s, got %s", p2p.PeerTypeProvider, p.Type)
		}
		<-done
	})
}
