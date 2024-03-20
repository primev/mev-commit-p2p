package preconfirmation_test

import (
	"context"
	"crypto/ecdh"
	"crypto/elliptic"
	"crypto/rand"
	"io"
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	p2ptest "github.com/primevprotocol/mev-commit/pkg/p2p/testing"
	"github.com/primevprotocol/mev-commit/pkg/preconfirmation"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfencryptor"
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

type testEncryptor struct {
	bidHash               []byte
	encryptedBid          *preconfencryptor.EncryptedBid
	bid                   *preconfencryptor.Bid
	preConfirmation       *preconfencryptor.EncryptedPreConfirmation
	bidSigner             common.Address
	preConfirmationSigner common.Address
}

func (t *testEncryptor) ConstructEncryptedBid(_ string, _ *big.Int, _ *big.Int) (*preconfencryptor.Bid, *preconfencryptor.EncryptedBid, error) {
	return t.bid, t.encryptedBid, nil
}

func (t *testEncryptor) ConstructEncryptedPreConfirmation(_ *preconfencryptor.Bid) (*preconfencryptor.EncryptedPreConfirmation, error) {
	return t.preConfirmation, nil
}

func (t *testEncryptor) VerifyBid(_ *preconfencryptor.Bid) (*common.Address, error) {
	return &t.bidSigner, nil
}

func (t *testEncryptor) DecryptBidData(_ common.Address, _ *preconfencryptor.EncryptedBid) (*preconfencryptor.Bid, error) {
	return t.bid, nil
}

func (t *testEncryptor) VerifyPreConfirmation(_ *preconfencryptor.PreConfirmation) (*common.Address, error) {
	return &t.preConfirmationSigner, nil
}

func (t *testEncryptor) VerifyEncryptedPreConfirmation(*ecdh.PublicKey, []byte, *preconfencryptor.EncryptedPreConfirmation) (*common.Address, error) {
	return &t.preConfirmationSigner, nil
}

type testProcessor struct {
	status providerapiv1.BidResponse_Status
}

func (t *testProcessor) ProcessBid(
	_ context.Context,
	_ *preconfencryptor.Bid) (chan providerapiv1.BidResponse_Status, error) {
	statusC := make(chan providerapiv1.BidResponse_Status, 1)
	statusC <- t.status
	return statusC, nil
}

type testCommitmentDA struct{}

func (t *testCommitmentDA) StoreEncryptedCommitment(
	_ context.Context,
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

		bid := &preconfencryptor.Bid{
			TxHash:      "test",
			BidAmt:      big.NewInt(10),
			BlockNumber: big.NewInt(10),
			Digest:      []byte("test"),
			Signature:   []byte("test"),
		}

		encryptedBid := &preconfencryptor.EncryptedBid{
			Ciphertext: []byte("test"),
		}

		// preConfirmation := &preconfencryptor.PreConfirmation{
		// 	Bid:       *bid,
		// 	Digest:    []byte("test"),
		// 	Signature: []byte("test"),
		// }

		encryptedPreConfirmation := &preconfencryptor.EncryptedPreConfirmation{
			Commitment: []byte("test"),
			Signature:  []byte("test"),
		}
		svc := p2ptest.New(
			&client,
		)

		topo := &testTopo{server}
		us := &testBidderStore{}
		proc := &testProcessor{
			status: providerapiv1.BidResponse_STATUS_ACCEPTED,
		}
		signer := &testEncryptor{
			bidHash:               bid.Digest,
			encryptedBid:          encryptedBid,
			bid:                   bid,
			preConfirmation:       encryptedPreConfirmation,
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

		respC, err := p.SendBid(context.Background(), bid.TxHash, bid.BidAmt, bid.BlockNumber)
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
