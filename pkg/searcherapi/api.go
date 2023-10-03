package searcherapi

import (
	"context"
	"log/slog"
	"math/big"

	searcherapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/searcherapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/preconf"
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
	SendBid(context.Context, string, *big.Int, *big.Int) (chan *preconf.PreconfCommitment, error)
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
		err := srv.Send(&searcherapiv1.Commitment{
			Bid: &searcherapiv1.Bid{
				TxnHash:     resp.TxnHash,
				BidAmt:      resp.Bid.Int64(),
				BlockNumber: resp.Blocknumber.Int64(),
			},
			BidHash:             resp.BidHash,
			Signature:           resp.Signature,
			DataHash:            resp.DataHash,
			CommitmentSignature: resp.CommitmentSignature,
		})
		if err != nil {
			s.logger.Error("error sending commitment", "err", err)
			return err
		}
		s.logger.Debug("sent commitment", "bid", resp.TxnHash)
	}

	return nil
}
