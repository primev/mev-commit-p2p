package allowancemanager

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	bidderregistry "github.com/primevprotocol/contracts-abi/clients/BidderRegistry"
	blocktracker "github.com/primevprotocol/contracts-abi/clients/BlockTracker"
	blocktrackercontract "github.com/primevprotocol/mev-commit/pkg/contracts/block_tracker"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	"github.com/primevprotocol/mev-commit/pkg/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BidderRegistry interface {
	CheckBidderAllowance(context.Context, common.Address, *big.Int, *big.Int) bool
	GetMinAllowance(ctx context.Context) (*big.Int, error)
}

type Store interface {
	GetBalance(bidder common.Address, windowNumber *big.Int) (*big.Int, error)
	SetBalance(bidder common.Address, windowNumber *big.Int, balance *big.Int) error
	DeductAndCheckBalanceForBlock(bidder common.Address, defaultAmount, bidAmount *big.Int, blockNumber int64) (*big.Int, error)
	RefundBalanceForBlock(bidder common.Address, amount *big.Int, blockNumber int64) error
}

type AllowanceManager struct {
	bidderRegistry  BidderRegistry
	blockTracker    blocktrackercontract.Interface
	commitmentDA    preconfcontract.Interface
	store           Store
	evtMgr          events.EventManager
	blocksPerWindow atomic.Uint64 // todo: move to the store
	minAllowance    atomic.Int64  // todo: move to the store
	currentWindow   atomic.Int64  // todo: move to the store
	logger          *slog.Logger
}

func NewAllowanceManager(
	br BidderRegistry,
	blockTracker blocktrackercontract.Interface,
	commitmentDA preconfcontract.Interface,
	store Store,
	evtMgr events.EventManager,
	logger *slog.Logger,
) *AllowanceManager {
	return &AllowanceManager{
		bidderRegistry: br,
		blockTracker:   blockTracker,
		commitmentDA:   commitmentDA,
		store:          store,
		evtMgr:         evtMgr,
		logger:         logger,
	}
}

func (a *AllowanceManager) Start(ctx context.Context) <-chan struct{} {
	doneChan := make(chan struct{})

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		ev1 := events.NewEventHandler(
			"NewWindow",
			func(window *blocktracker.BlocktrackerNewWindow) error {
				a.currentWindow.Store(window.Window.Int64())
				return nil
			},
		)

		sub1, err := a.evtMgr.Subscribe(ev1)
		if err != nil {
			return fmt.Errorf("failed to subscribe to NewWindow event: %w", err)
		}
		defer sub1.Unsubscribe()

		ev2 := events.NewEventHandler(
			"BidderRegistered",
			func(bidderReg *bidderregistry.BidderregistryBidderRegistered) error {
				// todo: do we need to check if commiter is connected to this bidder?
				return a.store.SetBalance(bidderReg.Bidder, bidderReg.WindowNumber, bidderReg.PrepaidAmount)
			},
		)

		sub2, err := a.evtMgr.Subscribe(ev2)
		if err != nil {
			return fmt.Errorf("failed to subscribe to BidderRegistered event: %w", err)
		}
		defer sub2.Unsubscribe()

		select {
		case <-egCtx.Done():
			return nil
		case err := <-sub1.Err():
			return fmt.Errorf("error in NewWindow event subscription: %w", err)
		case err := <-sub2.Err():
			return fmt.Errorf("error in BidderRegistered event subscription: %w", err)
		}
	})

	go func() {
		defer close(doneChan)
		if err := eg.Wait(); err != nil {
			a.logger.Error("error in AllowanceManager", "error", err)
		}
	}()

	return doneChan
}

func (a *AllowanceManager) CheckAndDeductAllowance(ctx context.Context, address common.Address, bidAmountStr string, blockNumber int64) (*big.Int, error) {
	if a.blocksPerWindow.Load() == 0 {
		blocksPerWindow, err := a.blockTracker.GetBlocksPerWindow(ctx)
		if err != nil {
			a.logger.Error("getting blocks per window", "error", err)
			return nil, status.Errorf(codes.Internal, "failed to get blocks per window: %v", err)
		}
		a.blocksPerWindow.Store(blocksPerWindow)
	}

	if a.minAllowance.Load() == 0 {
		minAllowance, err := a.bidderRegistry.GetMinAllowance(ctx)
		if err != nil {
			a.logger.Error("getting min allowance", "error", err)
			return nil, status.Errorf(codes.Internal, "failed to get min allowance: %v", err)

		}
		a.minAllowance.Store(minAllowance.Int64())
	}

	bidAmount, ok := new(big.Int).SetString(bidAmountStr, 10)
	if !ok {
		a.logger.Error("parsing bid amount", "amount", bidAmountStr)
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse bid amount")
	}

	// adding 2 to the current window, bcs oracle is 2 windows behind
	windowToCheck := big.NewInt(a.currentWindow.Load() + 2)

	balance, err := a.store.GetBalance(address, windowToCheck)
	if err != nil {
		a.logger.Error("getting balance", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get balance: %v", err)
	}

	if balance == nil {
		a.logger.Error("bidder balance not found", "address", address.Hex(), "window", windowToCheck)
		return nil, status.Errorf(codes.FailedPrecondition, "balance not found")
	}

	a.logger.Info("checking bidder allowance",
		"stake", balance.Uint64(),
		"blocksPerWindow", a.blocksPerWindow.Load(),
		"minStake", a.minAllowance.Load(),
		"window", windowToCheck.Uint64(),
		"address", address.Hex(),
	)

	blocksPerWindow := new(big.Int).SetUint64(a.blocksPerWindow.Load())
	minAllowance := big.NewInt(a.minAllowance.Load())

	// todo: make sense to do division only once, when bidder deposit funds,
	// not everytime, when checking allowance
	effectiveStake := new(big.Int).Div(new(big.Int).Set(balance), blocksPerWindow)

	isEnoughAllowance := effectiveStake.Cmp(minAllowance) >= 0

	if !isEnoughAllowance {
		a.logger.Error("bidder does not have enough allowance", "ethAddress", address)
		return nil, status.Errorf(codes.FailedPrecondition, "bidder not allowed")
	}

	deductedBalance, err := a.store.DeductAndCheckBalanceForBlock(address, effectiveStake, bidAmount, blockNumber)
	if err != nil {
		a.logger.Error("deducting balance", "error", err)
		return nil, status.Errorf(codes.FailedPrecondition, "failed to deduct balance: %v", err)
	}
	return deductedBalance, nil
}

func (a *AllowanceManager) RefundAllowance(address common.Address, deductedAmount *big.Int, blockNumber int64) error {
	return a.store.RefundBalanceForBlock(address, deductedAmount, blockNumber)
}
