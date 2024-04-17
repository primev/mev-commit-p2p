package events_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	bidderregistry "github.com/primevprotocol/contracts-abi/clients/BidderRegistry"
	"github.com/primevprotocol/mev-commit/pkg/events"
)

func TestEventHandler(t *testing.T) {
	t.Parallel()

	b := bidderregistry.BidderregistryBidderRegistered{
		Bidder:        common.HexToAddress("0xabcd"),
		PrepaidAmount: big.NewInt(1000),
		WindowNumber:  big.NewInt(99),
	}

	evtHdlr := events.NewEventHandler(
		"BidderRegistered",
		func(ev *bidderregistry.BidderregistryBidderRegistered) error {
			if ev.Bidder.Hex() != b.Bidder.Hex() {
				return fmt.Errorf("expected bidder %s, got %s", b.Bidder.Hex(), ev.Bidder.Hex())
			}
			if ev.PrepaidAmount.Cmp(b.PrepaidAmount) != 0 {
				return fmt.Errorf("expected prepaid amount %d, got %d", b.PrepaidAmount, ev.PrepaidAmount)
			}
			if ev.WindowNumber.Cmp(b.WindowNumber) != 0 {
				return fmt.Errorf("expected window number %d, got %d", b.WindowNumber, ev.WindowNumber)
			}
			return nil
		},
	)

	bidderABI, err := abi.JSON(strings.NewReader(bidderregistry.BidderregistryABI))
	if err != nil {
		t.Fatal(err)
	}

	event := bidderABI.Events["BidderRegistered"]

	evtHdlr.SetTopicAndContract(event.ID, &bidderABI)

	if evtHdlr.Topic().Cmp(event.ID) != 0 {
		t.Fatalf("expected topic %s, got %s", event.ID, evtHdlr.Topic())
	}

	if evtHdlr.EventName() != "BidderRegistered" {
		t.Fatalf("expected event name BidderRegistered, got %s", evtHdlr.EventName())
	}

	buf, err := event.Inputs.NonIndexed().Pack(
		b.PrepaidAmount,
		b.WindowNumber,
	)
	if err != nil {
		t.Fatal(err)
	}

	bidder := common.HexToHash(b.Bidder.Hex())

	// Creating a Log object
	testLog := types.Log{
		Topics: []common.Hash{
			event.ID, // The first topic is the hash of the event signature
			bidder,   // The next topics are the indexed event parameters
		},
		Data: buf,
	}

	if err := evtHdlr.Handle(testLog); err != nil {
		t.Fatal(err)
	}
}

func TestEventManager(t *testing.T) {
	t.Parallel()

	bidders := []bidderregistry.BidderregistryBidderRegistered{
		{
			Bidder:        common.HexToAddress("0xabcd"),
			PrepaidAmount: big.NewInt(1000),
			WindowNumber:  big.NewInt(99),
		},
		{
			Bidder:        common.HexToAddress("0xcdef"),
			PrepaidAmount: big.NewInt(2000),
			WindowNumber:  big.NewInt(100),
		},
	}

	count := 0

	handlerTriggered1 := make(chan struct{})
	handlerTriggered2 := make(chan struct{})

	evtHdlr := events.NewEventHandler(
		"BidderRegistered",
		func(ev *bidderregistry.BidderregistryBidderRegistered) error {
			if count >= len(bidders) {
				return fmt.Errorf("unexpected event")
			}
			if ev.Bidder.Hex() != bidders[count].Bidder.Hex() {
				return fmt.Errorf("expected bidder %s, got %s", bidders[count].Bidder.Hex(), ev.Bidder.Hex())
			}
			if ev.PrepaidAmount.Cmp(bidders[count].PrepaidAmount) != 0 {
				return fmt.Errorf("expected prepaid amount %d, got %d", bidders[count].PrepaidAmount, ev.PrepaidAmount)
			}
			if ev.WindowNumber.Cmp(bidders[count].WindowNumber) != 0 {
				return fmt.Errorf("expected window number %d, got %d", bidders[count].WindowNumber, ev.WindowNumber)
			}
			count++
			if count == 1 {
				close(handlerTriggered1)
			} else {
				close(handlerTriggered2)
			}
			return nil
		},
	)

	bidderABI, err := abi.JSON(strings.NewReader(bidderregistry.BidderregistryABI))
	if err != nil {
		t.Fatal(err)
	}

	data1, err := bidderABI.Events["BidderRegistered"].Inputs.NonIndexed().Pack(
		bidders[0].PrepaidAmount,
		bidders[0].WindowNumber,
	)
	if err != nil {
		t.Fatal(err)
	}

	data2, err := bidderABI.Events["BidderRegistered"].Inputs.NonIndexed().Pack(
		bidders[1].PrepaidAmount,
		bidders[1].WindowNumber,
	)
	if err != nil {
		t.Fatal(err)
	}

	evmClient := &testEVMClient{
		logs: []types.Log{
			{
				Topics: []common.Hash{
					bidderABI.Events["BidderRegistered"].ID,
					common.HexToHash(bidders[0].Bidder.Hex()),
				},
				Data:        data1,
				BlockNumber: 1,
			},
			{
				Topics: []common.Hash{
					bidderABI.Events["BidderRegistered"].ID,
					common.HexToHash(bidders[1].Bidder.Hex()),
				},
				Data:        data2,
				BlockNumber: 2,
			},
		},
	}

	store := &testStore{}

	contracts := map[common.Address]*abi.ABI{
		common.HexToAddress("0xabcd"): &bidderABI,
	}

	evtMgr := events.NewListener(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		evmClient,
		store,
		contracts,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := evtMgr.Start(ctx)

	sub, err := evtMgr.Subscribe(evtHdlr)
	if err != nil {
		t.Fatal(err)
	}

	defer sub.Unsubscribe()

	evmClient.SetBlockNumber(1)

	select {
	case <-handlerTriggered1:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler to be triggered")
	}

	evmClient.SetBlockNumber(2)
	select {
	case <-handlerTriggered2:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler to be triggered")
	}

	if b, err := store.LastBlock(); err != nil || b != 2 {
		t.Fatalf("expected block number 1, got %d", store.blockNumber)
	}

	cancel()
	<-done
}

type testEVMClient struct {
	mu       sync.Mutex
	blockNum uint64
	logs     []types.Log
}

func (t *testEVMClient) SetBlockNumber(blockNum uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.blockNum = blockNum
}

func (t *testEVMClient) BlockNumber(context.Context) (uint64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.blockNum, nil
}

func (t *testEVMClient) FilterLogs(
	ctx context.Context,
	q ethereum.FilterQuery,
) ([]types.Log, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	logs := make([]types.Log, 0, len(t.logs))
	for _, log := range t.logs {
		if log.BlockNumber >= q.FromBlock.Uint64() && log.BlockNumber <= q.ToBlock.Uint64() {
			logs = append(logs, log)
		}
	}

	return logs, nil
}

type testStore struct {
	mu          sync.Mutex
	blockNumber uint64
}

func (t *testStore) LastBlock() (uint64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.blockNumber, nil
}

func (t *testStore) SetLastBlock(blockNumber uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.blockNumber = blockNumber
	return nil
}