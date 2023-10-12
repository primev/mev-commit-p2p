package searcherapi

import (
	"context"
	"log/slog"
	"math/big"

	searcherapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/primevcrypto"
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
	SendBid(context.Context, string, *big.Int, *big.Int) (chan *primevcrypto.PreConfirmation, error)
}

func (s *Service) SendBid(
	bid *searcherapiv1.Bid,
	srv searcherapiv1.Searcher_SendBidServer,
) error {
	respC, err := s.sender.SendBid(
		srv.Context(),
		bid.TxnHash,
		big.NewInt(bid.BidAmt),
		big.NewInt(bid.BlockNumber),
	)
	if err != nil {
		s.logger.Error("error sending bid", "err", err)
		return status.Errorf(codes.Internal, "error sending bid: %v", err)
	}

	for resp := range respC {
		err := srv.Send(&searcherapiv1.PreConfirmation{
			Bid: &searcherapiv1.Bid{
				TxnHash:     resp.TxnHash,
				BidAmt:      resp.BidAmt.Int64(),
				BlockNumber: resp.BlockNumber.Int64(),
			},
			BidHash:                  resp.BidHash,
			Signature:                resp.Signature,
			PreconfirmationDigest:    resp.PreconfirmationDigest,
			PreConfirmationSignature: resp.PreConfirmationSignature,
		})
		if err != nil {
			s.logger.Error("error sending preConfirmation", "err", err)
			return err
		}
	}

	return nil
}
