package bidderapi

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/bufbuild/protovalidate-go"
	"github.com/ethereum/go-ethereum/common"
	bidderapiv1 "github.com/primevprotocol/mev-commit/gen/go/bidderapi/v1"
	preconfirmationv1 "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
	registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/bidder_registry"
	blocktrackercontract "github.com/primevprotocol/mev-commit/pkg/contracts/block_tracker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Service struct {
	bidderapiv1.UnimplementedBidderServer
	sender               PreconfSender
	owner                common.Address
	registryContract     registrycontract.Interface
	blockTrackerContract blocktrackercontract.Interface
	logger               *slog.Logger
	metrics              *metrics
	validator            *protovalidate.Validator
	depositedWindows     map[*big.Int]struct{}
}

func NewService(
	sender PreconfSender,
	owner common.Address,
	registryContract registrycontract.Interface,
	blockTrackerContract blocktrackercontract.Interface,
	validator *protovalidate.Validator,
	logger *slog.Logger,
) *Service {
	return &Service{
		sender:               sender,
		owner:                owner,
		registryContract:     registryContract,
		blockTrackerContract: blockTrackerContract,
		logger:               logger,
		metrics:              newMetrics(),
		validator:            validator,
		depositedWindows:     make(map[*big.Int]struct{}),
	}
}

type PreconfSender interface {
	SendBid(context.Context, string, string, int64, int64, int64) (chan *preconfirmationv1.PreConfirmation, error)
}

func (s *Service) SendBid(
	bid *bidderapiv1.Bid,
	srv bidderapiv1.Bidder_SendBidServer,
) error {
	// timeout to prevent hanging of bidder node if provider node is not responding
	ctx, cancel := context.WithTimeout(srv.Context(), 10*time.Second)
	defer cancel()

	s.metrics.ReceivedBidsCount.Inc()

	err := s.validator.Validate(bid)
	if err != nil {
		s.logger.Error("bid validation", "error", err)
		return status.Errorf(codes.InvalidArgument, "validating bid: %v", err)
	}

	txnsStr := strings.Join(bid.TxHashes, ",")

	respC, err := s.sender.SendBid(
		ctx,
		txnsStr,
		bid.Amount,
		bid.BlockNumber,
		bid.DecayStartTimestamp,
		bid.DecayEndTimestamp,
	)
	if err != nil {
		s.logger.Error("sending bid", "error", err)
		return status.Errorf(codes.Internal, "error sending bid: %v", err)
	}

	for resp := range respC {
		b := resp.Bid
		err := srv.Send(&bidderapiv1.Commitment{
			TxHashes:             strings.Split(b.TxHash, ","),
			BidAmount:            b.BidAmount,
			BlockNumber:          b.BlockNumber,
			ReceivedBidDigest:    hex.EncodeToString(b.Digest),
			ReceivedBidSignature: hex.EncodeToString(b.Signature),
			CommitmentDigest:     hex.EncodeToString(resp.Digest),
			CommitmentSignature:  hex.EncodeToString(resp.Signature),
			ProviderAddress:      common.Bytes2Hex(resp.ProviderAddress),
			DecayStartTimestamp:  b.DecayStartTimestamp,
			DecayEndTimestamp:    b.DecayEndTimestamp,
		})
		if err != nil {
			s.logger.Error("sending preConfirmation", "error", err)
			return err
		}
		s.metrics.ReceivedPreconfsCount.Inc()
	}

	return nil
}

func (s *Service) PrepayAllowance(
	ctx context.Context,
	r *bidderapiv1.PrepayRequest,
) (*bidderapiv1.PrepayResponse, error) {
	err := s.validator.Validate(r)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validating prepay request: %v", err)
	}

	currentWindow, err := s.blockTrackerContract.GetCurrentWindow(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting current window: %v", err)
	}

	windowToDeposit, err := s.calculateWindowToDeposit(ctx, r, currentWindow)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "calculating window to deposit: %v", err)
	}
	if _, ok := s.depositedWindows[windowToDeposit]; ok {
		return nil, status.Errorf(codes.FailedPrecondition, "allowance already pre-paid for window %d", windowToDeposit.Int64())
	}

	for window := range s.depositedWindows {
		if window.Cmp(new(big.Int).SetUint64(currentWindow)) < 0 {
			err := s.registryContract.WithdrawAllowance(ctx, window)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "withdrawing allowance: %v", err)
			}
			s.logger.Info("withdrew allowance", "window", window)
			delete(s.depositedWindows, window)
		}
	}

	amount, success := big.NewInt(0).SetString(r.Amount, 10)
	if !success {
		return nil, status.Errorf(codes.InvalidArgument, "parsing amount: %v", r.Amount)
	}

	err = s.registryContract.PrepayAllowanceForSpecificWindow(ctx, amount, windowToDeposit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prepaying allowance: %v", err)
	}

	stakeAmount, err := s.registryContract.GetAllowance(ctx, s.owner, windowToDeposit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting allowance: %v", err)
	}

	s.logger.Info("prepay successful", "amount", stakeAmount.String(), "window", windowToDeposit)
	s.depositedWindows[windowToDeposit] = struct{}{}

	return &bidderapiv1.PrepayResponse{Amount: stakeAmount.String(), WindowNumber: wrapperspb.UInt64(windowToDeposit.Uint64())}, nil
}

func (s *Service) calculateWindowToDeposit(ctx context.Context, r *bidderapiv1.PrepayRequest, currentWindow uint64) (*big.Int, error) {
	if r.WindowNumber != nil {
		// Directly use the specified window number if available.
		return new(big.Int).SetUint64(r.WindowNumber.Value), nil
	} else if r.BlockNumber != nil {
		// Calculate the window based on the block number.
		blocksPerWindow, err := s.blockTrackerContract.GetBlocksPerWindow(ctx)
		if err != nil {
			return nil, fmt.Errorf("getting window for block: %w", err)
		}
		return new(big.Int).SetUint64((r.BlockNumber.Value-1)/blocksPerWindow + 1), nil
	}
	// Default to two windows ahead of the current window if no specific block or window is given.
	// This is for the case where the oracle works 2 windows behind the current window.
	return new(big.Int).SetUint64(currentWindow + 2), nil
}

func (s *Service) GetAllowance(
	ctx context.Context,
	r *bidderapiv1.GetAllowanceRequest,
) (*bidderapiv1.PrepayResponse, error) {
	var (
		window uint64
		err    error
	)
	if r.WindowNumber == nil {
		window, err = s.blockTrackerContract.GetCurrentWindow(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "getting current window: %v", err)
		}
		// as oracle working 2 windows behind the current window, we add + 2 here
		window += 2
	} else {
		window = r.WindowNumber.Value
	}
	stakeAmount, err := s.registryContract.GetAllowance(ctx, s.owner, new(big.Int).SetUint64(window))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting allowance: %v", err)
	}

	return &bidderapiv1.PrepayResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetMinAllowance(
	ctx context.Context,
	_ *bidderapiv1.EmptyMessage,
) (*bidderapiv1.PrepayResponse, error) {
	stakeAmount, err := s.registryContract.GetMinAllowance(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting min allowance: %v", err)
	}

	return &bidderapiv1.PrepayResponse{Amount: stakeAmount.String()}, nil
}
