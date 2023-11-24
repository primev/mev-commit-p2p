package evmclient

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TxRequest struct {
	To        *common.Address
	CallData  []byte
	GasPrice  *big.Int
	GasLimit  uint64
	GasFeeCap *big.Int
	Value     *big.Int
}

func (t TxRequest) String() string {
	return fmt.Sprintf(
		"To: %s, CallData: %x, GasPrice: %d, GasLimit: %d, GasFeeCap: %d, Value: %d",
		t.To.String(),
		t.CallData,
		t.GasPrice,
		t.GasLimit,
		t.GasFeeCap,
		t.Value,
	)
}

type Interface interface {
	Send(ctx context.Context, tx *TxRequest) (common.Hash, error)
	WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	Call(ctx context.Context, tx *TxRequest) ([]byte, error)
}

type AtomicNonce uint64

func (a *AtomicNonce) IncrementAndReceive() uint64 {
	return atomic.AddUint64((*uint64)(a), 1)
}

type evmClient struct {
	sendMtx     sync.Mutex
	chainID     *big.Int
	ethClient   *ethclient.Client
	owner       common.Address
	signer      *ecdsa.PrivateKey
	logger      *slog.Logger
	atomicNonce AtomicNonce
}

func New(
	owner common.Address,
	signer *ecdsa.PrivateKey,
	ethClient *ethclient.Client,
	logger *slog.Logger,
) (Interface, error) {
	chainID, err := ethClient.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain id: %w", err)
	}

	nonce, err := ethClient.PendingNonceAt(context.Background(), owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	return &evmClient{
		chainID:     chainID,
		ethClient:   ethClient,
		owner:       owner,
		signer:      signer,
		logger:      logger,
		atomicNonce: AtomicNonce(nonce),
	}, nil
}

func (c *evmClient) newTx(ctx context.Context, req *TxRequest) (*types.Transaction, error) {
	var err error

	nonce := c.atomicNonce.IncrementAndReceive()

	if req.GasLimit == 0 {
		// if gas limit is not provided, estimate it
		req.GasLimit, err = c.ethClient.EstimateGas(ctx, ethereum.CallMsg{
			From:  c.owner,
			To:    req.To,
			Data:  req.CallData,
			Value: req.Value,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", err)
		}
	}

	gasTipCap, err := c.ethClient.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas tip cap: %w", err)
	}

	if req.GasPrice == nil {
		// if gas price is not provided, use the default one
		req.GasPrice, err = c.ethClient.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %w", err)
		}
	}

	gasFeeCap := new(big.Int).Add(gasTipCap, req.GasPrice)

	return types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		ChainID:   c.chainID,
		To:        req.To,
		Value:     req.Value,
		Gas:       req.GasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Data:      req.CallData,
	}), nil
}

func (c *evmClient) Send(ctx context.Context, tx *TxRequest) (common.Hash, error) {
	c.sendMtx.Lock()
	defer c.sendMtx.Unlock()

	txnData, err := c.newTx(ctx, tx)
	if err != nil {
		c.logger.Error("failed to create tx", "err", err)
		return common.Hash{}, err
	}

	signedTx, err := types.SignTx(txnData, types.NewLondonSigner(c.chainID), c.signer)
	if err != nil {
		c.logger.Error("failed to sign tx", "err", err)
		return common.Hash{}, fmt.Errorf("failed to sign tx: %w", err)
	}

	err = c.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		c.logger.Error("failed to send tx", "err", err)
		return common.Hash{}, err
	}

	c.logger.Info("sent txn", "tx", txnData, "txHash", signedTx.Hash().Hex())

	return signedTx.Hash(), nil
}

func (c *evmClient) WaitForReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {
	queryTicker := time.NewTicker(1 * time.Second)
	defer queryTicker.Stop()

	for {
		receipt, err := c.ethClient.TransactionReceipt(ctx, txHash)
		if err == nil {
			c.logger.Info("tx receipt", "txHash", txHash.Hex(), "status", receipt.Status)
			return receipt, nil
		}

		if errors.Is(err, ethereum.NotFound) {
			c.logger.Debug("tx not found", "txHash", txHash.Hex())
		} else {
			c.logger.Error("failed to get tx receipt", "txHash", txHash.Hex(), "err", err)
		}

		select {
		case <-ctx.Done():
			c.logger.Error("context cancelled", "txHash", txHash.Hex())
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

func (c *evmClient) Call(
	ctx context.Context,
	tx *TxRequest,
) ([]byte, error) {

	msg := ethereum.CallMsg{
		From:     c.owner,
		To:       tx.To,
		Data:     tx.CallData,
		Gas:      tx.GasLimit,
		GasPrice: tx.GasPrice,
		Value:    tx.Value,
	}

	result, err := c.ethClient.CallContract(ctx, msg, nil)
	if err != nil {
		c.logger.Error("failed to call contract", "err", err)
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	return result, nil
}
