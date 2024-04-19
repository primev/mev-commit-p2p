package blocktrackercontract

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	blocktracker "github.com/primevprotocol/contracts-abi/clients/BlockTracker"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
)

var blockTrackerABI = func() abi.ABI {
	abi, err := abi.JSON(strings.NewReader(blocktracker.BlocktrackerMetaData.ABI))
	if err != nil {
		panic(err)
	}
	return abi
}()

type Interface interface {
	// GetLastL1BlockNumber returns the number of the last L1 block recorded.
	GetLastL1BlockNumber(ctx context.Context) (uint64, error)
	// GetLastL1BlockWinner returns the winner of the last L1 block recorded.
	GetLastL1BlockWinner(ctx context.Context) (common.Address, error)
	// GetBlocksPerWindow returns the number of blocks per window.
	GetBlocksPerWindow(ctx context.Context) (uint64, error)
	// GetCurrentWindow returns the current window number.
	GetCurrentWindow(ctx context.Context) (uint64, error)
	// GetBlockWinner returns the winner of a specific block.
	GetBlockWinner(ctx context.Context, blockNumber uint64) (common.Address, error)
}

type blockTrackerContract struct {
	blockTrackerABI          abi.ABI
	blockTrackerContractAddr common.Address
	client                   evmclient.Interface
	wsClient                 evmclient.Interface
	logger                   *slog.Logger
}

type NewL1BlockEvent struct {
	BlockNumber *big.Int
	Winner      common.Address
	Window      *big.Int
}

func New(
	blockTrackerContractAddr common.Address,
	client evmclient.Interface,
	wsClient evmclient.Interface,
	logger *slog.Logger,
) Interface {
	return &blockTrackerContract{
		blockTrackerABI:          blockTrackerABI,
		blockTrackerContractAddr: blockTrackerContractAddr,
		client:                   client,
		wsClient:                 wsClient,
		logger:                   logger,
	}
}

// GetLastL1BlockNumber returns the number of the last L1 block recorded.
func (btc *blockTrackerContract) GetLastL1BlockNumber(ctx context.Context) (uint64, error) {
	callData, err := btc.blockTrackerABI.Pack("getLastL1BlockNumber")
	if err != nil {
		btc.logger.Error("error packing call data for getLastL1BlockNumber", "error", err)
		return 0, err
	}

	result, err := btc.client.Call(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return 0, err
	}

	results, err := btc.blockTrackerABI.Unpack("getLastL1BlockNumber", result)
	if err != nil {
		btc.logger.Error("error unpacking result for getLastL1BlockNumber", "error", err)
		return 0, err
	}

	lastBlockNumber, ok := results[0].(*big.Int)
	if !ok {
		return 0, fmt.Errorf("invalid result type")
	}

	return lastBlockNumber.Uint64(), nil
}

// GetLastL1BlockWinner returns the winner of the last L1 block recorded.
func (btc *blockTrackerContract) GetLastL1BlockWinner(ctx context.Context) (common.Address, error) {
	callData, err := btc.blockTrackerABI.Pack("getLastL1BlockWinner")
	if err != nil {
		btc.logger.Error("error packing call data for getLastL1BlockWinner", "error", err)
		return common.Address{}, err
	}

	result, err := btc.client.Call(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return common.Address{}, err
	}

	results, err := btc.blockTrackerABI.Unpack("getLastL1BlockWinner", result)
	if err != nil {
		btc.logger.Error("error unpacking result for getLastL1BlockWinner", "error", err)
		return common.Address{}, err
	}

	winnerAddress, ok := results[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid result type")
	}

	return winnerAddress, nil
}

// GetBlocksPerWindow returns the number of blocks per window.
func (btc *blockTrackerContract) GetBlocksPerWindow(ctx context.Context) (uint64, error) {
	callData, err := btc.blockTrackerABI.Pack("getBlocksPerWindow")
	if err != nil {
		btc.logger.Error("error packing call data for getBlocksPerWindow", "error", err)
		return 0, err
	}

	result, err := btc.client.Call(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return 0, err
	}

	results, err := btc.blockTrackerABI.Unpack("getBlocksPerWindow", result)
	if err != nil {
		btc.logger.Error("error unpacking result for getBlocksPerWindow", "error", err)
		return 0, err
	}

	blocksPerWindow, ok := results[0].(*big.Int)
	if !ok {
		return 0, fmt.Errorf("invalid result type")
	}

	return blocksPerWindow.Uint64(), nil
}

// GetCurrentWindow returns the current window number.
func (btc *blockTrackerContract) GetCurrentWindow(ctx context.Context) (uint64, error) {
	callData, err := btc.blockTrackerABI.Pack("getCurrentWindow")
	if err != nil {
		btc.logger.Error("error packing call data for getCurrentWindow", "error", err)
		return 0, err
	}

	result, err := btc.client.Call(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return 0, err
	}

	results, err := btc.blockTrackerABI.Unpack("getCurrentWindow", result)
	if err != nil {
		btc.logger.Error("error unpacking result for getCurrentWindow", "error", err)
		return 0, err
	}

	currentWindow, ok := results[0].(*big.Int)
	if !ok {
		return 0, fmt.Errorf("invalid result type")
	}

	return currentWindow.Uint64(), nil
}

// GetBlockWinner returns the winner of a specific block.
func (btc *blockTrackerContract) GetBlockWinner(ctx context.Context, blockNumber uint64) (common.Address, error) {
	callData, err := btc.blockTrackerABI.Pack("getBlockWinner", new(big.Int).SetUint64(blockNumber))
	if err != nil {
		btc.logger.Error("error packing call data for getBlockWinner", "error", err)
		return common.Address{}, err
	}

	result, err := btc.client.Call(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return common.Address{}, err
	}

	results, err := btc.blockTrackerABI.Unpack("getBlockWinner", result)
	if err != nil {
		btc.logger.Error("error unpacking result for getBlockWinner", "error", err)
		return common.Address{}, err
	}

	winnerAddress, ok := results[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid result type")
	}

	return winnerAddress, nil
}
