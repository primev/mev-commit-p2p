package preconfirmation

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	preconfpb "github.com/primevprotocol/mev-commit/gen/go/preconfirmation/v1"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/providerapi/v1"
	blocktrackercontract "github.com/primevprotocol/mev-commit/pkg/contracts/block_tracker"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	encryptor "github.com/primevprotocol/mev-commit/pkg/signer/preconfencryptor"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "1.0.0"
)

type EncryptedPreConfirmationWithDecrypted struct {
	*preconfpb.EncryptedPreConfirmation
	*preconfpb.PreConfirmation
}

type Preconfirmation struct {
	owner common.Address
	// todo: store the commitments in a database
	commitmentByBlockNumber map[int64][]*EncryptedPreConfirmationWithDecrypted
	encryptor               encryptor.Encryptor
	topo                    Topology
	streamer                p2p.Streamer
	us                      BidderStore
	processer               BidProcessor
	commitmentDA            preconfcontract.Interface
	blockTracker            blocktrackercontract.Interface
	logger                  *slog.Logger
	metrics                 *metrics
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

type BidderStore interface {
	CheckBidderAllowance(context.Context, common.Address, *big.Int, *big.Int) bool
}

type BidProcessor interface {
	ProcessBid(context.Context, *preconfpb.Bid) (chan providerapiv1.BidResponse_Status, error)
}

func New(
	owner common.Address,
	topo Topology,
	streamer p2p.Streamer,
	encryptor encryptor.Encryptor,
	us BidderStore,
	processor BidProcessor,
	commitmentDA preconfcontract.Interface,
	blockTracker blocktrackercontract.Interface,
	logger *slog.Logger,
) *Preconfirmation {
	commitmentsByBlockNumber := make(map[int64][]*EncryptedPreConfirmationWithDecrypted)
	return &Preconfirmation{
		owner:                   owner,
		commitmentByBlockNumber: commitmentsByBlockNumber,
		topo:                    topo,
		streamer:                streamer,
		encryptor:               encryptor,
		us:                      us,
		processer:               processor,
		commitmentDA:            commitmentDA,
		blockTracker:            blockTracker,
		logger:                  logger,
		metrics:                 newMetrics(),
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

			encryptedAndDecryptedPreconfirmation := &EncryptedPreConfirmationWithDecrypted{
				EncryptedPreConfirmation: encryptedPreConfirmation,
				PreConfirmation:          preConfirmation,
			}

			p.commitmentByBlockNumber[blockNumber] = append(p.commitmentByBlockNumber[blockNumber], encryptedAndDecryptedPreconfirmation)

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

	window, err := p.blockTracker.GetCurrentWindow(ctx)
	if err != nil {
		p.logger.Error("getting window", "error", err)
		return status.Errorf(codes.Internal, "failed to get window: %v", err)
	}

	blocksPerWindow, err := p.blockTracker.GetBlocksPerWindow(ctx)
	if err != nil {
		p.logger.Error("getting blocks per window", "error", err)
		return status.Errorf(codes.Internal, "failed to get blocks per window: %v", err)
	}

	if !p.us.CheckBidderAllowance(ctx, *ethAddress, new(big.Int).SetUint64(window), new(big.Int).SetUint64(blocksPerWindow)) {
		p.logger.Error("bidder does not have enough allowance", "ethAddress", ethAddress)
		return status.Errorf(codes.FailedPrecondition, "bidder not allowed")
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
			commitmentIndex, err := p.commitmentDA.StoreEncryptedCommitment(
				ctx,
				encryptedPreConfirmation.Commitment,
				encryptedPreConfirmation.Signature,
			)
			if err != nil {
				p.logger.Error("storing commitment", "error", err)
				return status.Errorf(codes.Internal, "failed to store commitments: %v", err)
			}

			encryptedPreConfirmation.CommitmentIndex = commitmentIndex.Bytes()
			encryptedAndDecryptedPreconfirmation := &EncryptedPreConfirmationWithDecrypted{
				EncryptedPreConfirmation: encryptedPreConfirmation,
				PreConfirmation:          preConfirmation,
			}
			blockNumber := preConfirmation.Bid.BlockNumber

			p.commitmentByBlockNumber[blockNumber] = append(p.commitmentByBlockNumber[blockNumber], encryptedAndDecryptedPreconfirmation)

			return stream.WriteMsg(ctx, encryptedPreConfirmation)
		}
	}
	return nil
}

func (p *Preconfirmation) StartListeningToNewL1BlockEvents(ctx context.Context, handler func(context.Context, blocktrackercontract.NewL1BlockEvent)) {
	ch := make(chan blocktrackercontract.NewL1BlockEvent)
	sub, err := p.blockTracker.SubscribeNewL1Block(ctx, ch) // Use ctx instead of context.Background()
	if err != nil {
		p.logger.Error("Failed to subscribe to NewL1Block events", "error", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case event := <-ch:
			handler(ctx, event) // Call the handler function
		case err := <-sub.Err():
			p.logger.Error("Subscription error", "error", err)
			return
		case <-ctx.Done(): // Handle cancellation
			p.logger.Info("Subscription context cancelled")
			return
		}
	}
}

func (p *Preconfirmation) HandleNewL1BlockEvent(ctx context.Context, event blocktrackercontract.NewL1BlockEvent) {
	p.logger.Info("New L1 Block event received", "blockNumber", event.BlockNumber, "winner", event.Winner, "window", event.Window)
	for _, commitment := range p.commitmentByBlockNumber[event.BlockNumber.Int64()] {
		_, err := p.commitmentDA.OpenCommitment(
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
			p.logger.Error("Failed to open commitment", "error", err)
			continue
		} else {
			p.logger.Info("Opened commitment", "txHash", commitment.PreConfirmation.Bid.TxHash)
		}
	}
	delete(p.commitmentByBlockNumber, event.BlockNumber.Int64())
}
