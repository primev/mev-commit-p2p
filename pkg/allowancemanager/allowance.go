package allowancemanager

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	bidderregistry "github.com/primevprotocol/contracts-abi/clients/BidderRegistry"
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
}

type AllowanceManager struct {
	bidderRegistry  BidderRegistry
	blockTracker    blocktrackercontract.Interface
	commitmentDA    preconfcontract.Interface
	store           Store
	evtMgr          events.EventManager
	blocksPerWindow *big.Int // todo: move to the store
	minAllowance    *big.Int // todo: move to the store
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
		return a.subscribeBidderRegistered(egCtx)
	})

	go func() {
		defer close(doneChan)
		if err := eg.Wait(); err != nil {
			a.logger.Error("error in AllowanceManager", "error", err)
		}
	}()

	return doneChan
}

func (a *AllowanceManager) CheckAllowance(ctx context.Context, address common.Address, window *big.Int) error {
	if a.blocksPerWindow == nil {
		blocksPerWindow, err := a.blockTracker.GetBlocksPerWindow(ctx)
		if err != nil {
			a.logger.Error("getting blocks per window", "error", err)
			return status.Errorf(codes.Internal, "failed to get blocks per window: %v", err)
		}
		a.blocksPerWindow = new(big.Int).SetUint64(blocksPerWindow)
	}

	if a.minAllowance == nil {
		minAllowance, err := a.bidderRegistry.GetMinAllowance(ctx)
		if err != nil {
			a.logger.Error("getting min allowance", "error", err)
			return status.Errorf(codes.Internal, "failed to get min allowance: %v", err)

		}
		a.minAllowance = minAllowance
	}

	balance, err := a.store.GetBalance(address, window)
	if err != nil {
		a.logger.Error("getting balance", "error", err)
		return status.Errorf(codes.Internal, "failed to get balance: %v", err)
	}

	a.logger.Info("checking bidder allowance",
		"stake", balance.Uint64(),
		"blocksPerWindow", a.blocksPerWindow,
		"minStake", a.minAllowance.Uint64(),
		"window", window.Uint64(),
		"address", address.Hex(),
	)
	
	isEnoughAllowance := (balance.Div(balance, a.blocksPerWindow)).Cmp(a.minAllowance) >= 0

	if !isEnoughAllowance {
		a.logger.Error("bidder does not have enough allowance", "ethAddress", address)
		return status.Errorf(codes.FailedPrecondition, "bidder not allowed")
	}

	return nil
}

func (a *AllowanceManager) subscribeBidderRegistered(ctx context.Context) error {
	ev := events.NewEventHandler(
		"BidderRegistered",
		func(bidderReg *bidderregistry.BidderregistryBidderRegistered) error {
			// todo: do we need to check if commiter is connected to this bidder?
			err := a.store.SetBalance(bidderReg.Bidder, bidderReg.WindowNumber, bidderReg.PrepaidAmount)
			if err != nil {
				return err
			}
			return nil
		},
	)

	sub, err := a.evtMgr.Subscribe(ev)
	if err != nil {
		return fmt.Errorf("failed to subscribe to BidderRegistered event: %w", err)
	}
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		return nil
	case err := <-sub.Err():
		return fmt.Errorf("error in BidderRegistered event subscription: %w", err)
	}
}
