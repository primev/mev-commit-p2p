package bidderapi

import (
	"context"
	"encoding/hex"
	"errors"
	bidderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/bidderapi/v1"
	"log/slog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/bidder_registry"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	bidderapiv1.UnimplementedBidderServer
	sender           PreconfSender
	owner            common.Address
	registryContract registrycontract.Interface
	logger           *slog.Logger
}

func NewService(
	sender PreconfSender,
	owner common.Address,
	registryContract registrycontract.Interface,
	logger *slog.Logger,
) *Service {
	return &Service{
		sender:           sender,
		owner:            owner,
		registryContract: registryContract,
		logger:           logger,
	}
}

type PreconfSender interface {
	SendBid(context.Context, string, *big.Int, *big.Int) (chan *preconfsigner.PreConfirmation, error)
}

func (s *Service) SendBid(
	bid *bidderapiv1.Bid,
	srv bidderapiv1.Bidder_SendBidServer,
) error {
	// timeout to prevent hanging of bidder node if provider node is not responding
	ctx, cancel := context.WithTimeout(srv.Context(), 10*time.Second)
	defer cancel()

	respC, err := s.sender.SendBid(
		ctx,
		bid.TxHash,
		big.NewInt(bid.Amount),
		big.NewInt(bid.BlockNumber),
	)
	if err != nil {
		s.logger.Error("error sending bid", "err", err)
		return status.Errorf(codes.Internal, "error sending bid: %v", err)
	}

	for resp := range respC {
		b := resp.Bid
		err := srv.Send(&bidderapiv1.Commitment{
			TxHash:               b.TxHash,
			BidAmount:            b.BidAmt.Int64(),
			BlockNumber:          b.BlockNumber.Int64(),
			ReceivedBidDigest:    hex.EncodeToString(b.Digest),
			ReceivedBidSignature: hex.EncodeToString(b.Signature),
			CommitmentDigest:     hex.EncodeToString(resp.Digest),
			CommitmentSignature:  hex.EncodeToString(resp.Signature),
		})
		if err != nil {
			s.logger.Error("error sending preConfirmation", "err", err)
			return err
		}
	}

	return nil
}

var ErrInvalidAmount = errors.New("invalid amount for stake")

func (s *Service) RegisterStake(
	ctx context.Context,
	stake *bidderapiv1.StakeRequest,
) (*bidderapiv1.StakeResponse, error) {
	amount, success := big.NewInt(0).SetString(stake.Amount, 10)
	if !success {
		return nil, ErrInvalidAmount
	}
	err := s.registryContract.RegisterBidder(ctx, amount)
	if err != nil {
		return nil, err
	}

	stakeAmount, err := s.registryContract.GetStake(ctx, s.owner)
	if err != nil {
		return nil, err
	}

	return &bidderapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetStake(
	ctx context.Context,
	_ *bidderapiv1.EmptyMessage,
) (*bidderapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetStake(ctx, s.owner)
	if err != nil {
		return nil, err
	}

	return &bidderapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetMinStake(
	ctx context.Context,
	_ *bidderapiv1.EmptyMessage,
) (*bidderapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetMinStake(ctx)
	if err != nil {
		return nil, err
	}

	return &bidderapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}
