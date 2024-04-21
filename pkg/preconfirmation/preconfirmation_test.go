package preconfirmation_test

import (
	"context"
	"crypto/ecdh"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	blocktracker "github.com/primevprotocol/contracts-abi/clients/BlockTracker"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/providerapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/events"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/store"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

type testTopo struct {
	peer p2p.Peer
}

func (t *testTopo) GetPeers(q topology.Query) []p2p.Peer {
	return []p2p.Peer{t.peer}
}

type testEncryptor struct {
	bidHash                  []byte
	encryptedBid             *preconfpb.EncryptedBid
	bid                      *preconfpb.Bid
	encryptedPreConfirmation *preconfpb.EncryptedPreConfirmation
	preConfirmation          *preconfpb.PreConfirmation
	sharedSecretKey          []byte
	bidSigner                common.Address
	preConfirmationSigner    common.Address
}

func (t *testEncryptor) ConstructEncryptedBid(_ string, _ string, _ int64, _ int64, _ int64) (*preconfpb.Bid, *preconfpb.EncryptedBid, error) {
	return t.bid, t.encryptedBid, nil
}

func (t *testEncryptor) ConstructEncryptedPreConfirmation(_ *preconfpb.Bid) (*preconfpb.PreConfirmation, *preconfpb.EncryptedPreConfirmation, error) {
	return t.preConfirmation, t.encryptedPreConfirmation, nil
}

func (t *testEncryptor) VerifyBid(_ *preconfpb.Bid) (*common.Address, error) {
	return &t.bidSigner, nil
}

func (t *testEncryptor) VerifyPreConfirmation(_ *preconfpb.PreConfirmation) (*common.Address, error) {
	return &t.preConfirmationSigner, nil
}

func (t *testEncryptor) DecryptBidData(_ common.Address, _ *preconfpb.EncryptedBid) (*preconfpb.Bid, error) {
	return t.bid, nil
}

func (t *testEncryptor) VerifyEncryptedPreConfirmation(*ecdh.PublicKey, []byte, *preconfpb.EncryptedPreConfirmation) ([]byte, *common.Address, error) {
	return t.sharedSecretKey, &t.preConfirmationSigner, nil
}

type testProcessor struct {
	status providerapiv1.BidResponse_Status
}

func (t *testProcessor) ProcessBid(
	_ context.Context,
	_ *preconfpb.Bid) (chan providerapiv1.BidResponse_Status, error) {
	statusC := make(chan providerapiv1.BidResponse_Status, 1)
	statusC <- t.status
	return statusC, nil
}

type testCommitmentDA struct{}

func (t *testCommitmentDA) StoreEncryptedCommitment(
	_ context.Context,
	_ []byte,
	_ []byte,
) (common.Hash, error) {
	return common.Hash{}, nil
}

func (t *testCommitmentDA) OpenCommitment(
	_ context.Context,
	_ []byte,
	_ string,
	_ int64,
	_ string,
	_ int64,
	_ int64,
	_ []byte,
	_ []byte,
	_ []byte,
) (common.Hash, error) {
	return common.Hash{}, nil
}

func (t *testCommitmentDA) Close() error {
	return nil
}

type testBlockTrackerContract struct {
	blockNumberToWinner map[uint64]common.Address
	lastBlockNumber     uint64
	blocksPerWindow     uint64
}

// GetCurrentWindow returns the current window number.
func (btc *testBlockTrackerContract) GetCurrentWindow(ctx context.Context) (uint64, error) {
	return btc.lastBlockNumber / btc.blocksPerWindow, nil
}

// GetBlocksPerWindow returns the number of blocks per window.
func (btc *testBlockTrackerContract) GetBlocksPerWindow(ctx context.Context) (uint64, error) {
	return btc.blocksPerWindow, nil
}

type testEventManager struct {
	btABI      *abi.ABI
	handler    events.EventHandler
	handlerSub chan struct{}
	sub        *testSub
}

type testSub struct {
	errC chan error
}

func (t *testSub) Unsubscribe() {}

func (t *testSub) Err() <-chan error {
	return t.errC
}

func (t *testEventManager) Subscribe(evt events.EventHandler) (events.Subscription, error) {
	if evt.EventName() != "NewL1Block" {
		return nil, errors.New("invalid event")
	}
	evt.SetTopicAndContract(t.btABI.Events["NewL1Block"].ID, t.btABI)
	t.handler = evt
	close(t.handlerSub)

	return t.sub, nil
}

func newTestLogger(t *testing.T, w io.Writer) *slog.Logger {
	t.Helper()

	testLogger := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(testLogger)
}

type testAllowanceManager struct{}

func (t *testAllowanceManager) Start(ctx context.Context) <-chan struct{} {
	return nil
}

func (t *testAllowanceManager) CheckAndDeductAllowance(ctx context.Context, address common.Address, bidAmountStr string, blockNumber int64) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (t *testAllowanceManager) RefundAllowance(address common.Address, deductedAmount *big.Int, blockNumber int64) error {
	return nil
}

func TestPreconfBidSubmission(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		client := p2p.Peer{
			EthAddress: common.HexToAddress("0x1"),
			Type:       p2p.PeerTypeBidder,
		}

		encryptionPrivateKey, err := ecies.GenerateKey(rand.Reader, elliptic.P256(), nil)
		if err != nil {
			t.Fatal(err)
		}

		nikePrivateKey, err := ecdh.P256().GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}

		server := p2p.Peer{
			EthAddress: common.HexToAddress("0x2"),
			Type:       p2p.PeerTypeProvider,
			Keys: &p2p.Keys{
				PKEPublicKey:  &encryptionPrivateKey.PublicKey,
				NIKEPublicKey: nikePrivateKey.PublicKey(),
			},
		}

		bid := &preconfpb.Bid{
			TxHash:              "test",
			BidAmount:           "10",
			BlockNumber:         10,
			DecayStartTimestamp: time.Now().UnixMilli() - 10000*time.Millisecond.Milliseconds(),
			DecayEndTimestamp:   time.Now().UnixMilli(),
			Digest:              []byte("test"),
			Signature:           []byte("test"),
		}

		encryptedBid := &preconfpb.EncryptedBid{
			Ciphertext: []byte("test"),
		}

		preConfirmation := &preconfpb.PreConfirmation{
			Bid:       bid,
			Digest:    []byte("test"),
			Signature: []byte("test"),
		}

		encryptedPreConfirmation := &preconfpb.EncryptedPreConfirmation{
			Commitment: []byte("test"),
			Signature:  []byte("test"),
		}
		svc := p2ptest.New(
			&client,
		)

		topo := &testTopo{server}
		proc := &testProcessor{
			status: providerapiv1.BidResponse_STATUS_ACCEPTED,
		}
		signer := &testEncryptor{
			bidHash:                  bid.Digest,
			encryptedBid:             encryptedBid,
			bid:                      bid,
			preConfirmation:          preConfirmation,
			encryptedPreConfirmation: encryptedPreConfirmation,
			bidSigner:                common.HexToAddress("0x1"),
			preConfirmationSigner:    common.HexToAddress("0x2"),
		}

		btABI, err := abi.JSON(strings.NewReader(blocktracker.BlocktrackerABI))
		if err != nil {
			t.Fatal(err)
		}
		eventManager := &testEventManager{
			btABI:      &btABI,
			sub:        &testSub{errC: make(chan error)},
			handlerSub: make(chan struct{}),
		}
		store, err := store.NewStore()
		if err != nil {
			t.Fatal(err)
		}
		allowanceMgr := &testAllowanceManager{}
		p := preconfirmation.New(
			client.EthAddress,
			topo,
			svc,
			signer,
			allowanceMgr,
			proc,
			&testCommitmentDA{},
			&testBlockTrackerContract{blockNumberToWinner: make(map[uint64]common.Address), blocksPerWindow: 64},
			eventManager,
			store,
			newTestLogger(t, os.Stdout),
		)

		svc.SetPeerHandler(server, p.Streams()[0])

		respC, err := p.SendBid(context.Background(), bid.TxHash, bid.BidAmount, bid.BlockNumber, bid.DecayStartTimestamp, bid.DecayEndTimestamp)
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
