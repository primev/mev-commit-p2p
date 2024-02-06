package evmclient

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// EVM is an interface for interacting with the Ethereum Virtual Machine. It includes
// subset of the methods from the go-ethereum ethclient.Client interface.
type EVM interface {
	// Client returns the underlying RPC client.
	Batcher() Batcher
	// NetworkID returns the network ID associated with this client.
	NetworkID(ctx context.Context) (*big.Int, error)
	// BlockNumber returns the most recent block number
	BlockNumber(ctx context.Context) (uint64, error)
	// PendingNonceAt retrieves the current pending nonce associated with an account.
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	// NonceAt retrieves the current nonce associated with an account.
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
	// execution of a transaction.
	// Note after eip 1559 this returns suggested priority fee per gas + suggested base fee per gas.
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	// SuggestGasTipCap retrieves the currently suggested 1559 priority fee to allow
	// a timely execution of a transaction.
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	// EstimateGas tries to estimate the gas needed to execute a specific
	// transaction based on the current pending state of the backend blockchain.
	// There is no guarantee that this is the true gas limit requirement as other
	// transactions may be added or removed by miners, but it should provide a basis
	// for setting a reasonable default.
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error)
	// SendTransaction injects the transaction into the pending pool for execution.
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	// ContractCall executes an Ethereum contract call with the specified data as the
	// input.
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	// TransactionReceipt returns the receipt of a transaction by transaction hash.
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	// TransactionByHash checks the pool of pending transactions in addition to the
	// blockchain. The isPending return value indicates whether the transaction has been
	// mined yet. Note that the transaction may not be part of the canonical chain even if
	// it's not pending.
	TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error)
}

type Batcher interface {
	// BatchCallContext executes multiple Ethereum contract calls concurrently with the
	// specified data as the input.
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type evm struct {
	*ethclient.Client
}

func WrapEthClient(client *ethclient.Client) EVM {
	return &evm{
		Client: client,
	}
}

func (e *evm) Batcher() Batcher {
	return e.Client.Client()
}
