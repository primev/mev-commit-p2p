package builderapi

import (
	"context"
	"log/slog"
	"sync"

	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/preconf"
)

type Service struct {
	builderapiv1.UnimplementedBuilderServer
	receiver      chan *builderapiv1.Bid
	bidsInProcess map[string]func(builderapiv1.BidResponse_Status)
	bidsMu        sync.Mutex
	logger        *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
	return &Service{
		receiver:      make(chan *builderapiv1.Bid),
		bidsInProcess: make(map[string]func(builderapiv1.BidResponse_Status)),
		logger:        logger,
	}
}

func (s *Service) ProcessBid(
	ctx context.Context,
	bid *preconf.Bid,
) (chan builderapiv1.BidResponse_Status, error) {
	respC := make(chan builderapiv1.BidResponse_Status, 1)
	s.bidsMu.Lock()
	s.bidsInProcess[string(bid.BidHash)] = func(status builderapiv1.BidResponse_Status) {
		respC <- status
		close(respC)
	}
	s.bidsMu.Unlock()

	select {
	case <-ctx.Done():
		s.bidsMu.Lock()
		delete(s.bidsInProcess, string(bid.BidHash))
		s.bidsMu.Unlock()

		return nil, ctx.Err()
	case s.receiver <- &builderapiv1.Bid{
		TxnHash:     bid.TxnHash,
		BidAmt:      bid.BidAmt.Int64(),
		BlockNumber: bid.BlockNumber.Int64(),
		BidHash:     bid.BidHash,
	}:
	}

	return respC, nil
}

func (s *Service) ReceiveBids(
	_ *builderapiv1.EmptyMessage,
	srv builderapiv1.Builder_ReceiveBidsServer,
) error {
	for {
		select {
		case <-srv.Context().Done():
			return srv.Context().Err()
		case bid := <-s.receiver:
			err := srv.Send(bid)
			if err != nil {
				return err
			}
		}
	}
}

func (s *Service) SendProcessedBids(srv builderapiv1.Builder_SendProcessedBidsServer) error {
	for {
		status, err := srv.Recv()
		if err != nil {
			return err
		}

		s.bidsMu.Lock()
		if callback, ok := s.bidsInProcess[string(status.BidHash)]; ok {
			callback(status.Status)
			delete(s.bidsInProcess, string(status.BidHash))
		}
		s.bidsMu.Unlock()
	}
}
