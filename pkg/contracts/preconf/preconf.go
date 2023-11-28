package preconfcontract

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	contractsabi "github.com/primevprotocol/mev-commit/pkg/abi"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
)

var preconfABI = func() abi.ABI {
	abi, err := abi.JSON(strings.NewReader(contractsabi.PreConfJson))
	if err != nil {
		panic(err)
	}
	return abi
}

var defaultWaitTimeout = 10 * time.Second

type Interface interface {
	StoreCommitment(
		ctx context.Context,
		bid *big.Int,
		blockNumber uint64,
		txHash string,
		bidSignature []byte,
		commitmentSignature []byte,
	) error

	io.Closer
}

type preconfContract struct {
	preconfABI          abi.ABI
	preconfContractAddr common.Address
	client              evmclient.Interface
	logger              *slog.Logger
	baseCtx             context.Context
	baseCtxCancel       context.CancelFunc
	txnWaitWg           sync.WaitGroup
}

func New(
	preconfContractAddr common.Address,
	client evmclient.Interface,
	logger *slog.Logger,
) Interface {
	baseCtx, baseCtxCancel := context.WithCancel(context.Background())
	return &preconfContract{
		preconfABI:          preconfABI(),
		preconfContractAddr: preconfContractAddr,
		client:              client,
		logger:              logger,
		baseCtx:             baseCtx,
		baseCtxCancel:       baseCtxCancel,
	}
}

func (p *preconfContract) StoreCommitment(
	ctx context.Context,
	bid *big.Int,
	blockNumber uint64,
	txHash string,
	bidSignature []byte,
	commitmentSignature []byte,
) error {

	callData, err := p.preconfABI.Pack(
		"storeCommitment",
		uint64(bid.Int64()),
		blockNumber,
		txHash,
		bidSignature,
		commitmentSignature,
	)
	if err != nil {
		p.logger.Error("preconf contract storeCommitment pack error", "err", err)
		return err
	}

	txnHash, err := p.client.Send(ctx, &evmclient.TxRequest{
		To:       &p.preconfContractAddr,
		CallData: callData,
	})
	if err != nil {
		return err
	}

	p.waitForTxnReceipt(txnHash)
	p.logger.Info("preconf contract storeCommitment successful", "txnHash", txnHash)

	return nil
}

func (p *preconfContract) Close() error {
	p.baseCtxCancel()
	done := make(chan struct{})
	go func() {
		p.txnWaitWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(defaultWaitTimeout):
		p.logger.Error("preconf contract close timeout")
		return errors.New("preconf contract close timeout")
	}
	return nil
}

func (p *preconfContract) waitForTxnReceipt(txnHash common.Hash) {
	p.txnWaitWg.Add(1)
	go func() {
		defer p.txnWaitWg.Done()

		logger := p.logger.With("txnHash", txnHash)

		ctx, cancel := context.WithTimeout(p.baseCtx, defaultWaitTimeout)
		defer cancel()

		receipt, err := p.client.WaitForReceipt(ctx, txnHash)
		switch err {
		case nil:
			if receipt.Status != types.ReceiptStatusSuccessful {
				logger.Error(
					"preconf contract storeCommitment receipt error", "receipt", receipt,
				)
				return
			}
			logger.Info("preconf storeCommitment receipt successful", "receipt", receipt)
		case context.Canceled:
			fallthrough
		case context.DeadlineExceeded:
			logger.Warn("cancelling preconf storeCommitment", "error", err)
			cancelHash, err := p.client.CancelTx(context.Background(), txnHash)
			if err != nil && !errors.Is(err, ethereum.NotFound) {
				logger.Error("preconf storeCommitment cancel error", "err", err)
				return
			}
			logger.Info("preconf storeCommitment canceled", "cancelHash", cancelHash)
		default:
			logger.Error("preconf contract storeCommitment receipt error", "err", err)
		}
	}()
}
