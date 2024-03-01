package bidderapi

import (
	"context"
	"encoding/hex"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/bufbuild/protovalidate-go"
	"github.com/ethereum/go-ethereum/common"
	bidderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/bidderapi/v1"
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
	metrics          *metrics
	validator        *protovalidate.Validator
}

func NewService(
	sender PreconfSender,
	owner common.Address,
	registryContract registrycontract.Interface,
	validator *protovalidate.Validator,
	logger *slog.Logger,
) *Service {
	return &Service{
		sender:           sender,
		owner:            owner,
		registryContract: registryContract,
		logger:           logger,
		metrics:          newMetrics(),
		validator:        validator,
	}
}

type PreconfSender interface {
	SendBid(context.Context, string, *big.Int, *big.Int, uint64, uint64) (chan *preconfsigner.PreConfirmation, error)
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

	amtVal, success := big.NewInt(0).SetString(bid.Amount, 10)
	if !success {
		s.logger.Error("parsing amount", "amount", bid.Amount)
		return status.Errorf(codes.InvalidArgument, "error parsing amount: %v", bid.Amount)
	}

	txnsStr := strings.Join(bid.TxHashes, ",")

	respC, err := s.sender.SendBid(
		ctx,
		txnsStr,
		amtVal,
		big.NewInt(bid.BlockNumber),
	)
	if err != nil {
		s.logger.Error("sending bid", "error", err)
		return status.Errorf(codes.Internal, "error sending bid: %v", err)
	}

	for resp := range respC {
		b := resp.Bid
		err := srv.Send(&bidderapiv1.Commitment{
			TxHashes:             strings.Split(b.TxHash, ","),
			BidAmount:            b.BidAmt.Int64(),
			BlockNumber:          b.BlockNumber.Int64(),
			ReceivedBidDigest:    hex.EncodeToString(b.Digest),
			ReceivedBidSignature: hex.EncodeToString(b.Signature),
			CommitmentDigest:     hex.EncodeToString(resp.Digest),
			CommitmentSignature:  hex.EncodeToString(resp.Signature),
			ProviderAddress:      resp.ProviderAddress.String(),
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
	stake *bidderapiv1.PrepayRequest,
) (*bidderapiv1.PrepayResponse, error) {
	err := s.validator.Validate(stake)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validating prepay request: %v", err)
	}

	amount, success := big.NewInt(0).SetString(stake.Amount, 10)
	if !success {
		return nil, status.Errorf(codes.InvalidArgument, "parsing amount: %v", stake.Amount)
	}

	err = s.registryContract.PrepayAllowance(ctx, amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prepaying allowance: %v", err)
	}

	stakeAmount, err := s.registryContract.GetAllowance(ctx, s.owner)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting allowance: %v", err)
	}

	return &bidderapiv1.PrepayResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetAllowance(
	ctx context.Context,
	_ *bidderapiv1.EmptyMessage,
) (*bidderapiv1.PrepayResponse, error) {
	stakeAmount, err := s.registryContract.GetAllowance(ctx, s.owner)
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
