package preconfcontract_test

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
	mockevmclient "github.com/primevprotocol/mev-commit/pkg/evmclient/mock"
	"github.com/primevprotocol/mev-commit/pkg/util"
)

func TestPreconfContract(t *testing.T) {
	t.Parallel()

	t.Run("StoreCommitment", func(t *testing.T) {
		preConfContract := common.HexToAddress("abcd")
		txHash := common.HexToHash("abcdef")
		bid := big.NewInt(1000000000000000000)
		blockNum := uint64(100)
		bidHash := "abcdef"
		bidSig := []byte("abcdef")
		commitment := []byte("abcdef")

		expCallData, err := preconfcontract.PreConfABI().Pack(
			"storeCommitment",
			uint64(bid.Int64()),
			blockNum,
			bidHash,
			bidSig,
			commitment,
		)

		if err != nil {
			t.Fatal(err)
		}

		mockClient := mockevmclient.New(
			mockevmclient.WithSendFunc(
				func(ctx context.Context, req *evmclient.TxRequest) (common.Hash, error) {
					if req.To.Cmp(preConfContract) != 0 {
						t.Fatalf(
							"expected to address to be %s, got %s",
							preConfContract.Hex(), req.To.Hex(),
						)
					}
					if !bytes.Equal(req.CallData, expCallData) {
						t.Fatalf("expected call data to be %x, got %x", expCallData, req.CallData)
					}
					return txHash, nil
				},
			),
			mockevmclient.WithWaitForReceiptFunc(
				func(ctx context.Context, txnHash common.Hash) (*types.Receipt, error) {
					if txnHash != txHash {
						t.Fatalf("expected tx hash to be %s, got %s", txHash.Hex(), txnHash.Hex())
					}
					return &types.Receipt{
						Status: 1,
					}, nil
				},
			),
		)

		preConfContractClient := preconfcontract.New(
			preConfContract,
			mockClient,
			util.NewTestLogger(os.Stdout),
		)

		err = preConfContractClient.StoreCommitment(
			context.Background(),
			bid,
			blockNum,
			bidHash,
			bidSig,
			commitment,
		)
		if err != nil {
			t.Fatal(err)
		}

		err = preConfContractClient.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("StoreCommitment cancel on timeout", func(t *testing.T) {
		defer preconfcontract.SetDefaultWaitTimeout(1 * time.Second)()

		preConfContract := common.HexToAddress("abcd")
		txHash := common.HexToHash("abcdef")
		bid := big.NewInt(1000000000000000000)
		blockNum := uint64(100)
		bidHash := "abcdef"
		bidSig := []byte("abcdef")
		commitment := []byte("abcdef")

		cancelled := make(chan struct{})

		mockClient := mockevmclient.New(
			mockevmclient.WithSendFunc(
				func(ctx context.Context, req *evmclient.TxRequest) (common.Hash, error) {
					return txHash, nil
				},
			),
			mockevmclient.WithWaitForReceiptFunc(
				func(ctx context.Context, txnHash common.Hash) (*types.Receipt, error) {
					<-ctx.Done()
					return nil, ctx.Err()
				},
			),
			mockevmclient.WithCancelFunc(
				func(ctx context.Context, txnHash common.Hash) (common.Hash, error) {
					if txnHash != txHash {
						t.Fatalf("expected tx hash to be %s, got %s", txHash.Hex(), txnHash.Hex())
					}
					close(cancelled)
					return common.HexToHash("ab"), nil
				},
			),
		)

		preConfContractClient := preconfcontract.New(
			preConfContract,
			mockClient,
			util.NewTestLogger(os.Stdout),
		)

		err := preConfContractClient.StoreCommitment(
			context.Background(),
			bid,
			blockNum,
			bidHash,
			bidSig,
			commitment,
		)
		if err != nil {
			t.Fatal(err)
		}

		select {
		case <-cancelled:
		case <-time.After(5 * time.Second):
			t.Fatal("expected cancel to be called")
		}

		err = preConfContractClient.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("error on close", func(t *testing.T) {
		defer preconfcontract.SetDefaultWaitTimeout(1 * time.Second)()

		preConfContract := common.HexToAddress("abcd")
		txHash := common.HexToHash("abcdef")
		bid := big.NewInt(1000000000000000000)
		blockNum := uint64(100)
		bidHash := "abcdef"
		bidSig := []byte("abcdef")
		commitment := []byte("abcdef")

		mockClient := mockevmclient.New(
			mockevmclient.WithSendFunc(
				func(ctx context.Context, req *evmclient.TxRequest) (common.Hash, error) {
					return txHash, nil
				},
			),
			mockevmclient.WithWaitForReceiptFunc(
				func(ctx context.Context, txnHash common.Hash) (*types.Receipt, error) {
					for {
						time.Sleep(1 * time.Second)
					}
				},
			),
		)

		preConfContractClient := preconfcontract.New(
			preConfContract,
			mockClient,
			util.NewTestLogger(os.Stdout),
		)

		err := preConfContractClient.StoreCommitment(
			context.Background(),
			bid,
			blockNum,
			bidHash,
			bidSig,
			commitment,
		)
		if err != nil {
			t.Fatal(err)
		}

		err = preConfContractClient.Close()
		if err == nil {
			t.Fatal("expected error on close")
		}
	})
}
