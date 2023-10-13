package searcherapi

import (
	"context"
	"encoding/hex"
	"log/slog"
	"math/big"

	searcherapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	searcherapiv1.UnimplementedSearcherServer
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
	bid *searcherapiv1.Bid,
	srv searcherapiv1.Searcher_SendBidServer,
) error {
	respC, err := s.sender.SendBid(
		srv.Context(),
		bid.TxnHash,
		big.NewInt(bid.Amount),
		big.NewInt(bid.BlockNumber),
	)
	if err != nil {
		s.logger.Error("error sending bid", "err", err)
		return status.Errorf(codes.Internal, "error sending bid: %v", err)
	}

	for resp := range respC {
		err := srv.Send(&searcherapiv1.PreConfirmation{
			Bid: &searcherapiv1.Bid{
				TxnHash:     resp.Bid.TxnHash,
				Amount:      resp.Bid.BidAmt.Int64(),
				BlockNumber: resp.Bid.BlockNumber.Int64(),
			},
			BidHash:      hex.EncodeToString(resp.Bid.Digest),
			BidSignature: hex.EncodeToString(resp.Bid.Signature),
			Digest:       hex.EncodeToString(resp.Digest),
			Signature:    hex.EncodeToString(resp.Signature),
		})
		if err != nil {
			s.logger.Error("error sending preConfirmation", "err", err)
			return err
		}
	}

	return nil
}
