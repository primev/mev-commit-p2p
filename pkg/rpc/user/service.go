package userapi

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	userapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/userapi/v1"
	registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/userregistry"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	userapiv1.UnimplementedUserServer
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
	bid *userapiv1.Bid,
	srv userapiv1.User_SendBidServer,
) error {
	respC, err := s.sender.SendBid(
		srv.Context(),
		bid.TxHash,
		big.NewInt(bid.Amount),
		big.NewInt(bid.BlockNumber),
	)
	if err != nil {
		s.logger.Error("error sending bid", "err", err)
		return status.Errorf(codes.Internal, "error sending bid: %v", err)
	}

	for resp := range respC {
		err := srv.Send(&userapiv1.PreConfirmation{
			TxHash:                   resp.Bid.TxHash,
			Amount:                   resp.Bid.BidAmt.Int64(),
			BlockNumber:              resp.Bid.BlockNumber.Int64(),
			BidDigest:                hex.EncodeToString(resp.Bid.Digest),
			BidSignature:             hex.EncodeToString(resp.Bid.Signature),
			PreConfirmationDigest:    hex.EncodeToString(resp.Digest),
			PreConfirmationSignature: hex.EncodeToString(resp.Signature),
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
	stake *userapiv1.StakeRequest,
) (*userapiv1.StakeResponse, error) {
	amount, success := big.NewInt(0).SetString(stake.Amount, 10)
	if !success {
		return nil, ErrInvalidAmount
	}
	err := s.registryContract.RegisterUser(ctx, amount)
	if err != nil {
		return nil, err
	}

	stakeAmount, err := s.registryContract.GetStake(ctx, s.owner)
	if err != nil {
		return nil, err
	}

	return &userapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetStake(
	ctx context.Context,
	_ *userapiv1.EmptyMessage,
) (*userapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetStake(ctx, s.owner)
	if err != nil {
		return nil, err
	}

	stakedAmt := stakeAmount.Div(stakeAmount, big.NewInt(1e18))
	return &userapiv1.StakeResponse{Amount: stakedAmt.String()}, nil
}

func (s *Service) GetMinStake(
	ctx context.Context,
	_ *userapiv1.EmptyMessage,
) (*userapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetMinStake(ctx)
	if err != nil {
		return nil, err
	}

	return &userapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}
