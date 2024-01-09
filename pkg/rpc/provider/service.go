package providerapi

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	registrycontract "github.com/primevprotocol/mev-commit/pkg/contracts/provider_registry"
	"github.com/primevprotocol/mev-commit/pkg/evmclient"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
)

type Service struct {
	providerapiv1.UnimplementedProviderServer
	receiver         chan *providerapiv1.Bid
	bidsInProcess    map[string]func(providerapiv1.BidResponse_Status)
	bidsMu           sync.Mutex
	logger           *slog.Logger
	owner            common.Address
	registryContract registrycontract.Interface
	evmClient        EvmClient
	metrics          *metrics
}

type EvmClient interface {
	PendingTxns() []evmclient.TxnInfo
	CancelTx(ctx context.Context, txHash common.Hash) (common.Hash, error)
}

func NewService(
	logger *slog.Logger,
	registryContract registrycontract.Interface,
	owner common.Address,
	e EvmClient,
) *Service {
	return &Service{
		receiver:         make(chan *providerapiv1.Bid),
		bidsInProcess:    make(map[string]func(providerapiv1.BidResponse_Status)),
		registryContract: registryContract,
		owner:            owner,
		logger:           logger,
		evmClient:        e,
		metrics:          newMetrics(),
	}
}

func toString(bid *providerapiv1.Bid) string {
	return fmt.Sprintf(
		"{TxHash: %s, BidAmount: %d, BlockNumber: %d, BidDigest: %x}",
		bid.TxHash, bid.BidAmount, bid.BlockNumber, bid.BidDigest,
	)
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
			s.logger.Info("received bid from node", "bid", toString(bid))
			err := srv.Send(bid)
			if err != nil {
				return err
			}
			s.metrics.BidsSentToProviderCount.Inc()
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
			s.logger.Info(
				"received bid status from node",
				"bidDigest", hex.EncodeToString(status.BidDigest),
				"status", status.Status.String(),
			)
			callback(status.Status)
			if status.Status == providerapiv1.BidResponse_STATUS_ACCEPTED {
				s.metrics.BidsAcceptedByProviderCount.Inc()
			} else {
				s.metrics.BidsRejectedByProviderCount.Inc()
			}
		}
	}
}

var ErrInvalidAmount = errors.New("invalid amount for stake")

func (s *Service) RegisterStake(
	ctx context.Context,
	stake *providerapiv1.StakeRequest,
) (*providerapiv1.StakeResponse, error) {
	amount, success := big.NewInt(0).SetString(stake.Amount, 10)
	if !success {
		return nil, ErrInvalidAmount
	}
	err := s.registryContract.RegisterProvider(ctx, amount)
	if err != nil {
		return nil, err
	}

	stakeAmount, err := s.registryContract.GetStake(ctx, s.owner)
	if err != nil {
		return nil, err
	}

	return &providerapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetStake(
	ctx context.Context,
	_ *providerapiv1.EmptyMessage,
) (*providerapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetStake(ctx, s.owner)
	if err != nil {
		return nil, err
	}

	return &providerapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetMinStake(
	ctx context.Context,
	_ *providerapiv1.EmptyMessage,
) (*providerapiv1.StakeResponse, error) {
	stakeAmount, err := s.registryContract.GetMinStake(ctx)
	if err != nil {
		return nil, err
	}

	return &providerapiv1.StakeResponse{Amount: stakeAmount.String()}, nil
}

func (s *Service) GetPendingTxns(
	ctx context.Context,
	_ *providerapiv1.EmptyMessage,
) (*providerapiv1.PendingTxnsResponse, error) {
	txns := s.evmClient.PendingTxns()

	txnsMsg := make([]*providerapiv1.TransactionInfo, len(txns))
	for i, txn := range txns {
		txnsMsg[i] = &providerapiv1.TransactionInfo{
			TxHash:  txn.Hash,
			Nonce:   int64(txn.Nonce),
			Created: txn.Created,
		}
	}

	return &providerapiv1.PendingTxnsResponse{PendingTxns: txnsMsg}, nil
}

func (s *Service) CancelTransaction(
	ctx context.Context,
	cancel *providerapiv1.CancelReq,
) (*providerapiv1.CancelResponse, error) {
	txHash := common.HexToHash(cancel.TxHash)
	cHash, err := s.evmClient.CancelTx(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return &providerapiv1.CancelResponse{TxHash: cHash.Hex()}, nil
}
