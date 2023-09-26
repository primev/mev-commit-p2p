package handshake_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/libp2p/internal/handshake"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
)

type testRegister struct{}

func (t *testRegister) GetStake(_ common.Address) (*big.Int, error) {
	return big.NewInt(5), nil
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

		hs1 := handshake.New(
			privKey1,
			common.HexToAddress("0x1"),
			p2p.PeerTypeBuilder,
			"test",
			&testSigner{address: common.HexToAddress("0x2")},
			&testRegister{},
			big.NewInt(4),
			func(p core.PeerID) (common.Address, error) {
				return common.HexToAddress("0x2"), nil
			},
		)
		hs2 := handshake.New(
			privKey2,
			common.HexToAddress("0x2"),
			p2p.PeerTypeBuilder,
			"test",
			&testSigner{address: common.HexToAddress("0x1")},
			&testRegister{},
			big.NewInt(4),
			func(p core.PeerID) (common.Address, error) {
				return common.HexToAddress("0x1"), nil
			},
		)

		out, in := p2ptest.NewDuplexStream()

		done := make(chan struct{})
		go func() {
			defer close(done)

			p, err := hs1.Handle(context.Background(), in, core.PeerID("test2"))
			if err != nil {
				t.Fatal(err)
			}
			if p.EthAddress != common.HexToAddress("0x2") {
				t.Fatalf(
					"expected eth address %s, got %s",
					common.HexToAddress("0x2"), p.EthAddress,
				)
			}
			if p.Type != p2p.PeerTypeBuilder {
				t.Fatalf("expected peer type %s, got %s", p2p.PeerTypeBuilder, p.Type)
			}
		}()

		p, err := hs2.Handshake(context.Background(), core.PeerID("test1"), out)
		if err != nil {
			t.Fatal(err)
		}
		if p.EthAddress != common.HexToAddress("0x1") {
			t.Fatalf("expected eth address %s, got %s", common.HexToAddress("0x1"), p.EthAddress)
		}
		if p.Type != p2p.PeerTypeBuilder {
			t.Fatalf("expected peer type %s, got %s", p2p.PeerTypeBuilder, p.Type)
		}
		<-done
	})
}
