package preconfirmation_test

import (
	"context"
	"io"
	"log/slog"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

type testTopo struct {
	peer p2p.Peer
}

func (t *testTopo) GetPeers(q topology.Query) []p2p.Peer {
	return []p2p.Peer{t.peer}
}

type testBidderStore struct{}

func (t *testBidderStore) CheckBidderAllowance(_ context.Context, _ common.Address) bool {
	return true
}

type testSigner struct {
	bid                   *preconfsigner.Bid
	preConfirmation       *preconfsigner.PreConfirmation
	bidSigner             common.Address
	preConfirmationSigner common.Address
}

func (t *testSigner) ConstructSignedBid(_ string, _ *big.Int, _ *big.Int, _ *big.Int, _ *big.Int) (*preconfsigner.Bid, error) {
	return t.bid, nil
}

func (t *testSigner) ConstructPreConfirmation(_ *preconfsigner.Bid) (*preconfsigner.PreConfirmation, error) {
	return t.preConfirmation, nil
}

func (t *testSigner) VerifyBid(_ *preconfsigner.Bid) (*common.Address, error) {
	return &t.bidSigner, nil
}

func (t *testSigner) VerifyPreConfirmation(_ *preconfsigner.PreConfirmation) (*common.Address, error) {
	return &t.preConfirmationSigner, nil
}

type testProcessor struct {
	status providerapiv1.BidResponse_Status
}

func (t *testProcessor) ProcessBid(
	_ context.Context,
	_ *preconfsigner.Bid) (chan providerapiv1.BidResponse_Status, error) {
	statusC := make(chan providerapiv1.BidResponse_Status, 1)
	statusC <- t.status
	return statusC, nil
}

type testCommitmentDA struct{}

func (t *testCommitmentDA) StoreCommitment(
	_ context.Context,
	_ *big.Int,
	_ uint64,
	_ string,
	_ uint64,
	_ uint64,
	_ []byte,
	_ []byte,
) error {
	return nil
}

func (t *testCommitmentDA) Close() error {
	return nil
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
			Type:       p2p.PeerTypeBidder,
		}
		server := p2p.Peer{
			EthAddress: common.HexToAddress("0x2"),
			Type:       p2p.PeerTypeProvider,
		}

		bid := &preconfsigner.Bid{
			TxHash:              "test",
			BidAmt:              big.NewInt(10),
			BlockNumber:         big.NewInt(10),
			DecayStartTimeStamp: big.NewInt(time.Now().UnixMilli() - 10000*time.Millisecond.Milliseconds()),
			DecayEndTimeStamp:   big.NewInt(time.Now().UnixMilli()),
			Digest:              []byte("test"),
			Signature:           []byte("test"),
		}

		preConfirmation := &preconfsigner.PreConfirmation{
			Bid:       *bid,
			Digest:    []byte("test"),
			Signature: []byte("test"),
		}

		svc := p2ptest.New(
			&client,
		)

		topo := &testTopo{server}
		us := &testBidderStore{}
		proc := &testProcessor{
			status: providerapiv1.BidResponse_STATUS_ACCEPTED,
		}
		signer := &testSigner{
			bid:                   bid,
			preConfirmation:       preConfirmation,
			bidSigner:             common.HexToAddress("0x1"),
			preConfirmationSigner: common.HexToAddress("0x2"),
		}

		p := preconfirmation.New(
			topo,
			svc,
			signer,
			us,
			proc,
			&testCommitmentDA{},
			newTestLogger(t, os.Stdout),
		)

		svc.SetPeerHandler(server, p.Protocol())

		respC, err := p.SendBid(context.Background(), bid.TxHash, bid.BidAmt, bid.BlockNumber, bid.DecayStartTimeStamp, bid.DecayEndTimeStamp)
		if err != nil {
			t.Fatal(err)
		}

		commitment := <-respC

		if string(commitment.Digest) != "test" {
			t.Fatalf("data hash is not equal to test")
		}

		if string(commitment.Signature) != "test" {
			t.Fatalf("preConfirmation signature is not equal to test")
		}
	})
}
