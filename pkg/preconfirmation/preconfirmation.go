package preconfirmation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	blocktracker "github.com/primevprotocol/contracts-abi/clients/BlockTracker"
	preconfcommstore "github.com/primevprotocol/contracts-abi/clients/PreConfCommitmentStore"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/providerapi/v1"
	blocktrackercontract "github.com/primevprotocol/mev-commit/pkg/contracts/block_tracker"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	"github.com/primevprotocol/mev-commit/pkg/events"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	encryptor "github.com/primevprotocol/mev-commit/pkg/signer/preconfencryptor"
	"github.com/primevprotocol/mev-commit/pkg/store"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "3.0.0"
)

type Preconfirmation struct {
	owner        common.Address
	encryptor    encryptor.Encryptor
	topo         Topology
	streamer     p2p.Streamer
	allowanceMgr AllowanceManager
	processer    BidProcessor
	commitmentDA preconfcontract.Interface
	blockTracker blocktrackercontract.Interface
	evtMgr       events.EventManager
	ecds         EncrDecrCommitmentStore
	logger       *slog.Logger
	metrics      *metrics
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

type BidProcessor interface {
	ProcessBid(context.Context, *preconfpb.Bid) (chan providerapiv1.BidResponse_Status, error)
}

type EncrDecrCommitmentStore interface {
	GetCommitmentsByBlockNumber(blockNum int64) ([]*store.EncryptedPreConfirmationWithDecrypted, error)
	GetCommitmentByHash(commitmentHash string) (*store.EncryptedPreConfirmationWithDecrypted, error)
	AddCommitment(commitment *store.EncryptedPreConfirmationWithDecrypted)
	DeleteCommitmentByBlockNumber(blockNum int64) error
}

type AllowanceManager interface {
	Start(ctx context.Context) <-chan struct{}
	CheckAllowance(ctx context.Context, ethAddress common.Address, window *big.Int) error
}

func New(
	owner common.Address,
	topo Topology,
	streamer p2p.Streamer,
	encryptor encryptor.Encryptor,
	// us BidderStore,
	allowanceMgr AllowanceManager,
	processor BidProcessor,
	commitmentDA preconfcontract.Interface,
	blockTracker blocktrackercontract.Interface,
	evtMgr events.EventManager,
	edcs EncrDecrCommitmentStore,
	logger *slog.Logger,
) *Preconfirmation {
	return &Preconfirmation{
		owner:     owner,
		topo:      topo,
		streamer:  streamer,
		encryptor: encryptor,
		// us:           us,
		allowanceMgr: allowanceMgr,
		processer:    processor,
		commitmentDA: commitmentDA,
		blockTracker: blockTracker,
		evtMgr:       evtMgr,
		ecds:         edcs,
		logger:       logger,
		metrics:      newMetrics(),
	}
}

func (p *Preconfirmation) bidStream() p2p.StreamDesc {
	return p2p.StreamDesc{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		Handler: p.handleBid,
	}
}

func (p *Preconfirmation) Streams() []p2p.StreamDesc {
	return []p2p.StreamDesc{p.bidStream()}
}

func (p *Preconfirmation) Start(ctx context.Context) <-chan struct{} {
	doneChan := make(chan struct{})

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return p.subscribeNewL1Block(egCtx)
	})

	eg.Go(func() error {
		return p.subscribeEncryptedCommitmentStored(egCtx)
	})

	go func() {
		defer close(doneChan)
		if err := eg.Wait(); err != nil {
			p.logger.Error("failed to start preconfirmation", "error", err)
		}
	}()

	return doneChan
}

// SendBid is meant to be called by the bidder to construct and send bids to the provider.
// It takes the txHash, the bid amount in wei and the maximum valid block number.
// It waits for preConfirmations from all providers and then returns.
// It returns an error if the bid is not valid.
func (p *Preconfirmation) SendBid(
	ctx context.Context,
	txHash string,
	bidAmt string,
	blockNumber int64,
	decayStartTimestamp int64,
	decayEndTimestamp int64,
) (chan *preconfpb.PreConfirmation, error) {
	bid, encryptedBid, err := p.encryptor.ConstructEncryptedBid(txHash, bidAmt, blockNumber, decayStartTimestamp, decayEndTimestamp)
	if err != nil {
		p.logger.Error("constructing encrypted bid", "error", err, "txHash", txHash)
		return nil, err
	}
	p.logger.Info("constructed encrypted bid", "encryptedBid", encryptedBid)

	providers := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeProvider})
	if len(providers) == 0 {
		p.logger.Error("no providers available", "txHash", txHash)
		return nil, errors.New("no providers available")
	}

	// Create a new channel to receive preConfirmations
	preConfirmations := make(chan *preconfpb.PreConfirmation, len(providers))

	wg := sync.WaitGroup{}
	for idx := range providers {
		wg.Add(1)
		go func(provider p2p.Peer) {
			defer wg.Done()

			logger := p.logger.With("provider", provider, "bid", txHash)

			providerStream, err := p.streamer.NewStream(
				ctx,
				provider,
				nil,
				p.bidStream(),
			)
			if err != nil {
				logger.Error("creating stream", "error", err)
				return
			}

			logger.Info("sending encrypted bid", "encryptedBid", encryptedBid)

			err = providerStream.WriteMsg(ctx, encryptedBid)
			if err != nil {
				_ = providerStream.Reset()
				logger.Error("writing message", "error", err)
				return
			}
			p.metrics.SentBidsCount.Inc()

			encryptedPreConfirmation := new(preconfpb.EncryptedPreConfirmation)
			err = providerStream.ReadMsg(ctx, encryptedPreConfirmation)
			if err != nil {
				_ = providerStream.Reset()
				logger.Error("reading message", "error", err)
				return
			}

			_ = providerStream.Close()

			// Process preConfirmation as a bidder
			sharedSecretKey, providerAddress, err := p.encryptor.VerifyEncryptedPreConfirmation(provider.Keys.NIKEPublicKey, bid.Digest, encryptedPreConfirmation)
			if err != nil {
				logger.Error("verifying provider signature", "error", err)
				return
			}

			preConfirmation := &preconfpb.PreConfirmation{
				Bid:          bid,
				SharedSecret: sharedSecretKey,
				Digest:       encryptedPreConfirmation.Commitment,
				Signature:    encryptedPreConfirmation.Signature,
			}

			preConfirmation.ProviderAddress = make([]byte, len(providerAddress))
			copy(preConfirmation.ProviderAddress, providerAddress[:])

			encryptedAndDecryptedPreconfirmation := &store.EncryptedPreConfirmationWithDecrypted{
				EncryptedPreConfirmation: encryptedPreConfirmation,
				PreConfirmation:          preConfirmation,
			}

			p.ecds.AddCommitment(encryptedAndDecryptedPreconfirmation)
			logger.Info("received preconfirmation", "preConfirmation", preConfirmation)
			p.metrics.ReceivedPreconfsCount.Inc()

			select {
			case preConfirmations <- preConfirmation:
			case <-ctx.Done():
				logger.Error("context cancelled", "error", ctx.Err())
				return
			}
		}(providers[idx])
	}

	go func() {
		wg.Wait()
		close(preConfirmations)
	}()

	return preConfirmations, nil
}

var ErrInvalidBidderTypeForBid = errors.New("invalid bidder type for bid")

// handlebid is the function that is called when a bid is received
// It is meant to be used by the provider exclusively to read the bid value from the bidder.
func (p *Preconfirmation) handleBid(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {
	if peer.Type != p2p.PeerTypeBidder {
		return ErrInvalidBidderTypeForBid
	}

	encryptedBid := new(preconfpb.EncryptedBid)
	err := stream.ReadMsg(ctx, encryptedBid)
	if err != nil {
		return err
	}

	p.logger.Info("received bid", "encryptedBid", encryptedBid)
	bid, err := p.encryptor.DecryptBidData(peer.EthAddress, encryptedBid)
	if err != nil {
		return err
	}
	ethAddress, err := p.encryptor.VerifyBid(bid)
	if err != nil {
		return err
	}

	// todo: move to the event listening to allowance manager
	window, err := p.blockTracker.GetCurrentWindow(ctx)
	if err != nil {
		p.logger.Error("getting window", "error", err)
		return status.Errorf(codes.Internal, "failed to get window: %v", err)
	}

	err = p.allowanceMgr.CheckAllowance(ctx, *ethAddress, new(big.Int).SetUint64(window))
	if err != nil {
		p.logger.Error("checking allowance", "error", err)
		return err
	}
	// try to enqueue for 5 seconds
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	statusC, err := p.processer.ProcessBid(ctx, bid)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case st := <-statusC:
		switch st {
		case providerapiv1.BidResponse_STATUS_REJECTED:
			return status.Errorf(codes.Internal, "bid rejected")
		case providerapiv1.BidResponse_STATUS_ACCEPTED:
			preConfirmation, encryptedPreConfirmation, err := p.encryptor.ConstructEncryptedPreConfirmation(bid)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to constuct encrypted preconfirmation: %v", err)
			}
			p.logger.Info("sending preconfirmation", "preConfirmation", encryptedPreConfirmation)
			_, err = p.commitmentDA.StoreEncryptedCommitment(
				ctx,
				encryptedPreConfirmation.Commitment,
				encryptedPreConfirmation.Signature,
			)
			if err != nil {
				p.logger.Error("storing commitment", "error", err)
				return status.Errorf(codes.Internal, "failed to store commitments: %v", err)
			}

			encryptedAndDecryptedPreconfirmation := &store.EncryptedPreConfirmationWithDecrypted{
				EncryptedPreConfirmation: encryptedPreConfirmation,
				PreConfirmation:          preConfirmation,
			}

			p.ecds.AddCommitment(encryptedAndDecryptedPreconfirmation)

			return stream.WriteMsg(ctx, encryptedPreConfirmation)
		}
	}
	return nil
}

func (p *Preconfirmation) subscribeNewL1Block(ctx context.Context) error {
	ev := events.NewEventHandler(
		"NewL1Block",
		func(newL1Block *blocktracker.BlocktrackerNewL1Block) error {
			p.logger.Info("New L1 Block event received", "blockNumber", newL1Block.BlockNumber, "winner", newL1Block.Winner, "window", newL1Block.Window)
			commitments, err := p.ecds.GetCommitmentsByBlockNumber(newL1Block.BlockNumber.Int64())
			if err != nil {
				p.logger.Error("failed to get commitments by block number", "error", err)
				return err
			}
			for _, commitment := range commitments {
				if common.BytesToAddress(commitment.ProviderAddress) != newL1Block.Winner {
					p.logger.Info("provider address does not match the winner", "providerAddress", commitment.ProviderAddress, "winner", newL1Block.Winner)
					continue
				}
				txHash, err := p.commitmentDA.OpenCommitment(
					ctx,
					commitment.EncryptedPreConfirmation.CommitmentIndex,
					commitment.PreConfirmation.Bid.BidAmount,
					commitment.PreConfirmation.Bid.BlockNumber,
					commitment.PreConfirmation.Bid.TxHash,
					commitment.PreConfirmation.Bid.DecayStartTimestamp,
					commitment.PreConfirmation.Bid.DecayEndTimestamp,
					commitment.PreConfirmation.Bid.Signature,
					commitment.PreConfirmation.Signature,
					commitment.PreConfirmation.SharedSecret,
				)
				if err != nil {
					// todo: retry mechanism?
					p.logger.Error("failed to open commitment", "error", err)
					continue
				} else {
					p.logger.Info("opened commitment", "txHash", txHash)
				}
			}
			err = p.ecds.DeleteCommitmentByBlockNumber(newL1Block.BlockNumber.Int64())
			if err != nil {
				p.logger.Error("failed to delete commitments by block number", "error", err)
				return err
			}
			return nil
		},
	)

	sub, err := p.evtMgr.Subscribe(ev)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NewL1Block event: %w", err)
	}
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		return nil
	case err := <-sub.Err():
		return fmt.Errorf("subscription error: %w", err)
	}
}

func (p *Preconfirmation) subscribeEncryptedCommitmentStored(ctx context.Context) error {
	ev := events.NewEventHandler(
		"EncryptedCommitmentStored",
		func(ec *preconfcommstore.PreconfcommitmentstoreEncryptedCommitmentStored) error {
			p.logger.Info("Encrypted Commitment Stored event received", "commitmentDigest", ec.CommitmentDigest, "commitmentIndex", ec.CommitmentIndex)
			commitment, err := p.ecds.GetCommitmentByHash(common.Bytes2Hex(ec.CommitmentDigest[:]))
			if err != nil {
				return fmt.Errorf("failed to get commitment by hash: %w", err)
			}
			if commitment == nil {
				p.logger.Debug("commitment not found", "commitmentDigest", ec.CommitmentDigest)
				return nil
			}
			commitment.EncryptedPreConfirmation.CommitmentIndex = ec.CommitmentIndex[:]
			return nil
		},
	)

	sub, err := p.evtMgr.Subscribe(ev)
	if err != nil {
		return fmt.Errorf("failed to subscribe to EncryptedCommitmentStored event: %w", err)
	}
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
		return nil
	case err := <-sub.Err():
		return fmt.Errorf("encrypted commitment stored subscription error: %w", err)
	}
}
