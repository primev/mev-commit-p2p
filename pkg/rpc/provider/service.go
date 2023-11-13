package providerapi

import (
	"context"
	"log/slog"
	"math/big"
	"sync"

	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/registry"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
)

type Service struct {
	providerapiv1.UnimplementedProviderServer
	receiver         chan *providerapiv1.Bid
	bidsInProcess    map[string]func(providerapiv1.BidResponse_Status)
	bidsMu           sync.Mutex
	logger           *slog.Logger
	registryContract registrycontract.Interface
}

func NewService(logger *slog.Logger, registryContract registrycontract.Interface) *Service {
	return &Service{
		receiver:         make(chan *providerapiv1.Bid),
		bidsInProcess:    make(map[string]func(providerapiv1.BidResponse_Status)),
		registryContract: registryContract,
		logger:           logger,
	}
}

func (s *Service) ProcessBid(
	ctx context.Context,
	bid *preconfsigner.Bid,
) (chan providerapiv1.BidResponse_Status, error) {
	respC := make(chan providerapiv1.BidResponse_Status, 1)
	s.bidsMu.Lock()
	s.bidsInProcess[string(bid.Digest)] = func(status providerapiv1.BidResponse_Status) {
		respC <- status
		close(respC)
	}
	s.bidsMu.Unlock()

	select {
	case <-ctx.Done():
		s.bidsMu.Lock()
		delete(s.bidsInProcess, string(bid.Digest))
		s.bidsMu.Unlock()

		s.logger.Error("context cancelled for sending bid", "err", ctx.Err())
		return nil, ctx.Err()
	case s.receiver <- &providerapiv1.Bid{
		TxHash:      bid.TxHash,
		BidAmount:   bid.BidAmt.Int64(),
		BlockNumber: bid.BlockNumber.Int64(),
		BidDigest:   bid.Digest,
	}:
	}
	s.logger.Info("sent bid to provider node", "bid", bid)

	return respC, nil
}

func (s *Service) ReceiveBids(
	_ *providerapiv1.EmptyMessage,
	srv providerapiv1.Provider_ReceiveBidsServer,
) error {
	for {
		select {
		case <-srv.Context().Done():
			s.logger.Error("context cancelled for receiving bid", "err", srv.Context().Err())
			return srv.Context().Err()
		case bid := <-s.receiver:
			s.logger.Info("received bid from node", "bid", bid)
			err := srv.Send(bid)
			if err != nil {
				return err
			}
		}
	}
}

func (s *Service) SendProcessedBids(srv providerapiv1.Provider_SendProcessedBidsServer) error {
	for {
		status, err := srv.Recv()
		if err != nil {
			s.logger.Error("error receiving bid status", "err", err)
			return err
		}

		s.bidsMu.Lock()
		callback, ok := s.bidsInProcess[string(status.BidDigest)]
		delete(s.bidsInProcess, string(status.BidDigest))
		s.bidsMu.Unlock()

		if ok {
			s.logger.Info("received bid status from node", "status", status)
			callback(status.Status)
		}
	}
}

func (s *Service) RegisterStake(
	ctx context.Context,
	stake *providerapiv1.StakeRequest,
) (*providerapiv1.StakeResponse, error) {
	err := s.registryContract.RegisterProvider(ctx, big.NewInt(stake.Amount))
	if err != nil {
		return nil, err
	}

	stakeAmount, err := s.registryContract.GetStake(ctx)
	if err != nil {
		return nil, err
	}

	return &providerapiv1.StakeResponse{Amount: stakeAmount.Int64()}, nil
}

func (s *Service) GetStake(
	ctx context.Context,
	_ *providerapiv1.EmptyMessage,
) (*providerapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetStake(ctx)
	if err != nil {
		return nil, err
	}

	return &providerapiv1.StakeResponse{Amount: stakeAmount.Int64()}, nil
}
