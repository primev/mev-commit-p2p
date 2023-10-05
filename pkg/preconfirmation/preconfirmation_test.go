package preconfirmation_test

import (
	"context"
	"io"
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconf"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

type testTopo struct {
	peer p2p.Peer
}

func (t *testTopo) GetPeers(q topology.Query) []p2p.Peer {
	return []p2p.Peer{t.peer}
}

type testUserStore struct{}

func (t *testUserStore) CheckUserRegistred(_ *common.Address) bool {
	return true
}

type testSigner struct {
	bid              *preconf.Bid
	commitment       *preconf.Commitment
	bidSigner        common.Address
	commitmentSigner common.Address
}

func (t *testSigner) ConstructSignedBid(_ string, _ *big.Int, _ *big.Int) (*preconf.Bid, error) {
	return t.bid, nil
}

func (t *testSigner) ConstructCommitment(_ *preconf.Bid) (*preconf.Commitment, error) {
	return t.commitment, nil
}

func (t *testSigner) VerifyBid(_ *preconf.Bid) (*common.Address, error) {
	return &t.bidSigner, nil
}

func (t *testSigner) VerifyCommitment(_ *preconf.Commitment) (*common.Address, error) {
	return &t.commitmentSigner, nil
}

type testProcessor struct {
	status builderapiv1.BidResponse_Status
}

func (t *testProcessor) ProcessBid(
	_ context.Context,
	_ *preconf.Bid) (chan builderapiv1.BidResponse_Status, error) {
	statusC := make(chan builderapiv1.BidResponse_Status, 1)
	statusC <- t.status
	return statusC, nil
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

		bid := &preconf.Bid{
			TxnHash:     "test",
			BidAmt:      big.NewInt(10),
			BlockNumber: big.NewInt(10),
			BidHash:     []byte("test"),
			Signature:   []byte("test"),
		}

		commitment := &preconf.Commitment{
			Bid:                 *bid,
			DataHash:            []byte("test"),
			CommitmentSignature: []byte("test"),
		}

		svc := p2ptest.New(
			&client,
		)

		topo := &testTopo{server}
		us := &testUserStore{}
		proc := &testProcessor{
			status: builderapiv1.BidResponse_STATUS_ACCEPTED,
		}
		signer := &testSigner{
			bid:              bid,
			commitment:       commitment,
			bidSigner:        common.HexToAddress("0x1"),
			commitmentSigner: common.HexToAddress("0x2"),
		}

		p := preconfirmation.New(topo, svc, signer, us, proc, newTestLogger(t, os.Stdout))

		svc.SetPeerHandler(server, p.Protocol())

		respC, err := p.SendBid(context.Background(), bid.TxnHash, bid.BidAmt, bid.BlockNumber)
		if err != nil {
			t.Fatal(err)
		}

		resp := <-respC

		if string(resp.DataHash) != "test" {
			t.Fatalf("data hash is not equal to test")
		}

		if string(resp.CommitmentSignature) != "test" {
			t.Fatalf("commitment signature is not equal to test")
		}
	})
}
