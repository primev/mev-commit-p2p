package blocktrackercontract

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	// RecordL1Block records a new L1 block and its winner.
	RecordL1Block(ctx context.Context, blockNumber uint64, winner common.Address) error
	// GetLastL1BlockNumber returns the number of the last L1 block recorded.
	GetLastL1BlockNumber(ctx context.Context) (uint64, error)
	// GetLastL1BlockWinner returns the winner of the last L1 block recorded.
	GetLastL1BlockWinner(ctx context.Context) (common.Address, error)
	// GetBlocksPerWindow returns the number of blocks per window.
	GetBlocksPerWindow(ctx context.Context) (uint64, error)
	// SetBlocksPerWindow sets the number of blocks per window.
	SetBlocksPerWindow(ctx context.Context, blocksPerWindow uint64) error
	// GetCurrentWindow returns the current window number.
	GetCurrentWindow(ctx context.Context) (uint64, error)
	// GetBlockWinner returns the winner of a specific block.
	GetBlockWinner(ctx context.Context, blockNumber uint64) (common.Address, error)
	// SubscribeNewL1Block subscribes to the NewL1Block events emitted by the contract.
	SubscribeNewL1Block(ctx context.Context, eventCh chan<- NewL1BlockEvent) (ethereum.Subscription, error)
	// PollNewL1BlockEvents polls for NewL1Block events and sends them to the event channel.
	// PollNewL1BlockEvents(ctx context.Context, eventCh chan<- NewL1BlockEvent, pollInterval time.Duration) error
}

type blockTrackerContract struct {
	blockTrackerABI          abi.ABI
	blockTrackerContractAddr common.Address
	client                   evmclient.Interface
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
	logger *slog.Logger,
) Interface {
	return &blockTrackerContract{
		blockTrackerABI:          blockTrackerABI,
		blockTrackerContractAddr: blockTrackerContractAddr,
		client:                   client,
		logger:                   logger,
	}
}

// RecordL1Block records a new L1 block and its winner.
func (btc *blockTrackerContract) RecordL1Block(ctx context.Context, blockNumber uint64, winner common.Address) error {
	callData, err := btc.blockTrackerABI.Pack("recordL1Block", new(big.Int).SetUint64(blockNumber), winner)
	if err != nil {
		btc.logger.Error("error packing call data for recordL1Block", "error", err)
		return err
	}

	txnHash, err := btc.client.Send(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return err
	}

	receipt, err := btc.client.WaitForReceipt(ctx, txnHash)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		btc.logger.Error("recordL1Block transaction failed", "txnHash", txnHash, "receipt", receipt)
		return err
	}

	btc.logger.Info("recordL1Block transaction successful", "txnHash", txnHash)
	return nil
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

// SetBlocksPerWindow sets the number of blocks per window.
func (btc *blockTrackerContract) SetBlocksPerWindow(ctx context.Context, blocksPerWindow uint64) error {
	callData, err := btc.blockTrackerABI.Pack("setBlocksPerWindow", new(big.Int).SetUint64(blocksPerWindow))
	if err != nil {
		btc.logger.Error("error packing call data for setBlocksPerWindow", "error", err)
		return err
	}

	txnHash, err := btc.client.Send(ctx, &evmclient.TxRequest{
		To:       &btc.blockTrackerContractAddr,
		CallData: callData,
	})
	if err != nil {
		return err
	}

	receipt, err := btc.client.WaitForReceipt(ctx, txnHash)
	if err != nil {
		return err
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		btc.logger.Error("setBlocksPerWindow transaction failed", "txnHash", txnHash, "receipt", receipt)
		return fmt.Errorf("transaction failed with hash: %s", txnHash.Hex())
	}

	return nil
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

// SubscribeNewL1Block subscribes to the NewL1Block events emitted by the contract.
func (btc *blockTrackerContract) SubscribeNewL1Block(ctx context.Context, eventCh chan<- NewL1BlockEvent) (ethereum.Subscription, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{btc.blockTrackerContractAddr},
		Topics:    [][]common.Hash{{blockTrackerABI.Events["NewL1Block"].ID}},
	}

	logsCh := make(chan types.Log)
	sub, err := btc.client.SubscribeFilterLogs(ctx, query, logsCh)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case log := <-logsCh:
				event := NewL1BlockEvent{}
				err := blockTrackerABI.UnpackIntoInterface(&event, "NewL1Block", log.Data)
				if err != nil {
					btc.logger.Error("error unpacking NewL1Block event", "error", err)
					continue
				}
				event.BlockNumber = new(big.Int).SetBytes(log.Topics[1].Bytes())
				event.Winner = common.HexToAddress(log.Topics[2].Hex())
				event.Window = new(big.Int).SetBytes(log.Topics[3].Bytes())
				eventCh <- event
			case <-ctx.Done():
				sub.Unsubscribe()
				return
			}
		}
	}()

	return sub, nil
}

// func (btc *blockTrackerContract) PollNewL1BlockEvents(ctx context.Context, eventCh chan<- NewL1BlockEvent, pollInterval time.Duration) error {
// 	ticker := time.NewTicker(pollInterval)
// 	defer ticker.Stop()

// 	startBlock := uint64(0) // todo: take this variable from config

// 	for {
// 		select {
// 		case <-ticker.C:
// 			// Update the query to search for events from startBlock to the latest block
// 			query := ethereum.FilterQuery{
// 				FromBlock: big.NewInt(int64(startBlock)),
// 				Addresses: []common.Address{btc.blockTrackerContractAddr},
// 				Topics:    [][]common.Hash{{blockTrackerABI.Events["NewL1Block"].ID}},
// 			}

// 			logs, err := btc.client.FilterLogs(ctx, query)
// 			if err != nil {
// 				btc.logger.Error("error filtering NewL1Block events", "error", err)
// 				continue
// 			}

// 			for _, log := range logs {
// 				event := NewL1BlockEvent{}
// 				err := blockTrackerABI.UnpackIntoInterface(&event, "NewL1Block", log.Data)
// 				if err != nil {
// 					btc.logger.Error("error unpacking NewL1Block event", "error", err)
// 					continue
// 				}
// 				event.BlockNumber = new(big.Int).SetBytes(log.Topics[1].Bytes())
// 				event.Winner = common.HexToAddress(log.Topics[2].Hex())
// 				event.Window = new(big.Int).SetBytes(log.Topics[3].Bytes())

// 				eventCh <- event
// 			}

// 			// Update startBlock for the next query to start from the latest checked block
// 			if len(logs) > 0 {
// 				lastLog := logs[len(logs)-1]
// 				startBlock = lastLog.BlockNumber + 1
// 			}

// 		case <-ctx.Done():
// 			return ctx.Err()
// 		}
// 	}
// }
