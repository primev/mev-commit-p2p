package preconfirmation_test

import (
	"context"
	"io"
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

type testTopo struct{}

func (t *testTopo) GetPeers(q topology.Query) []p2p.Peer {
	return []p2p.Peer{{EthAddress: common.HexToAddress("0x2"), Type: p2p.PeerTypeBuilder}}
}

type testUserStore struct{}

func (t *testUserStore) CheckUserRegistred(_ *common.Address) bool {
	return true
}

func newTestLogger(t *testing.T, w io.Writer) *slog.Logger {
	t.Helper()

	testLogger := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(testLogger)
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
			&client,
		)

		topo := &testTopo{}
		us := &testUserStore{}
		key, _ := crypto.GenerateKey()
		p := preconfirmation.New(topo, svc, key, us, newTestLogger(t, os.Stdout))

		svc.SetPeerHandler(server, p.Protocol())

		respC, err := p.SendBid(
			context.Background(),
			"0x4c03a845396b770ad41b975d6bd3bf8c2bd5cca36867a3301f9598f2e3e9518d",
			big.NewInt(10),
			big.NewInt(10),
		)
		if err != nil {
			t.Fatal(err)
		}

		resp := <-respC

		if resp.DataHash == nil {
			t.Fatal("datahash is nil")
		}

		if resp.CommitmentSignature == nil {
			t.Fatal("commitment signature is nil")
		}
	})
}
