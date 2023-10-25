package userapi

import (
	"context"
	"encoding/hex"
	"log/slog"
	"math/big"

	userapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/userapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	userapiv1.UnimplementedUserServer
	sender PreconfSender
	logger *slog.Logger
}

func NewService(sender PreconfSender, logger *slog.Logger) *Service {
	return &Service{
		sender: sender,
		logger: logger,
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
