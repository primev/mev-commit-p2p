package preconfirmation

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	providerapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/providerapi/v1"
	preconfcontract "github.com/primevprotocol/mev-commit/pkg/contracts/preconf"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	encryptor "github.com/primevprotocol/mev-commit/pkg/signer/preconfencryptor"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "1.0.0"
)

type Preconfirmation struct {
	encryptor    encryptor.Encryptor
	topo         Topology
	streamer     p2p.Streamer
	us           BidderStore
	processer    BidProcessor
	commitmentDA preconfcontract.Interface
	logger       *slog.Logger
	metrics      *metrics
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

type BidderStore interface {
	CheckBidderAllowance(context.Context, common.Address) bool
}

type BidProcessor interface {
	ProcessBid(context.Context, *encryptor.Bid) (chan providerapiv1.BidResponse_Status, error)
}

func New(
	topo Topology,
	streamer p2p.Streamer,
	encryptor encryptor.Encryptor,
	us BidderStore,
	processor BidProcessor,
	commitmentDA preconfcontract.Interface,
	logger *slog.Logger,
) *Preconfirmation {
	return &Preconfirmation{
		topo:         topo,
		streamer:     streamer,
		encryptor:    encryptor,
		us:           us,
		processer:    processor,
		commitmentDA: commitmentDA,
		logger:       logger,
		metrics:      newMetrics(),
	}
}

func (p *Preconfirmation) Protocol() p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{
				Name:    "bid",
				Handler: p.handleBid,
			},
		},
	}
}

// SendBid is meant to be called by the bidder to construct and send bids to the provider.
// It takes the txHash, the bid amount in wei and the maximum valid block number.
// It waits for preConfirmations from all providers and then returns.
// It returns an error if the bid is not valid.
func (p *Preconfirmation) SendBid(
	ctx context.Context,
	txHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
) (chan *encryptor.PreConfirmation, error) {
	bid, signedBid, err := p.encryptor.ConstructEncryptedBid(txHash, bidAmt, blockNumber)
	if err != nil {
		p.logger.Error("constructing signed bid", "error", err, "txHash", txHash)
		return nil, err
	}
	p.logger.Info("constructed signed bid", "signedBid", signedBid)

	providers := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeProvider})
	if len(providers) == 0 {
		p.logger.Error("no providers available", "txHash", txHash)
		return nil, errors.New("no providers available")
	}

	// Create a new channel to receive preConfirmations
	preConfirmations := make(chan *encryptor.PreConfirmation, len(providers))

	wg := sync.WaitGroup{}
	for idx := range providers {
		wg.Add(1)
		go func(provider p2p.Peer) {
			defer wg.Done()

			logger := p.logger.With("provider", provider, "bid", txHash)

			providerStream, err := p.streamer.NewStream(
				ctx,
				provider,
				ProtocolName,
				ProtocolVersion,
				"bid",
			)
			if err != nil {
				logger.Error("creating stream", "error", err)
				return
			}

			logger.Info("sending signed bid", "signedBid", signedBid)

			r, w := msgpack.NewReaderWriter[encryptor.EncryptedPreConfirmation, encryptor.EncryptedBid](providerStream)
			err = w.WriteMsg(ctx, signedBid)
			if err != nil {
				_ = providerStream.Reset()
				logger.Error("writing message", "error", err)
				return
			}
			p.metrics.SentBidsCount.Inc()

			encryptedPreConfirmation, err := r.ReadMsg(ctx)
			if err != nil {
				_ = providerStream.Reset()
				logger.Error("reading message", "error", err)
				return
			}

			_ = providerStream.Close()

			// Process preConfirmation as a bidder
			providerAddress, err := p.encryptor.VerifyEncryptedPreConfirmation(provider.Keys.NIKEPublicKey, bid.Digest, encryptedPreConfirmation)
			if err != nil {
				logger.Error("verifying provider signature", "error", err)
				return
			}

			preConfirmation := &encryptor.PreConfirmation{
				Bid:             *bid,
				Digest:          encryptedPreConfirmation.Commitment,
				Signature:       encryptedPreConfirmation.Signature,
				ProviderAddress: *providerAddress,
			}

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

	r, w := msgpack.NewReaderWriter[encryptor.EncryptedBid, encryptor.EncryptedPreConfirmation](stream)
	encryptedBid, err := r.ReadMsg(ctx)
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

	// todo: change to take care of double spend
	if p.us.CheckBidderAllowance(ctx, *ethAddress) {
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
		case status := <-statusC:
			switch status {
			case providerapiv1.BidResponse_STATUS_REJECTED:
				return errors.New("bid rejected")
			case providerapiv1.BidResponse_STATUS_ACCEPTED:
				preConfirmation, err := p.encryptor.ConstructEncryptedPreConfirmation(bid)
				if err != nil {
					return err
				}
				p.logger.Info("sending preconfirmation", "preConfirmation", preConfirmation)
				// todo: update SC
				err = p.commitmentDA.StoreEncryptedCommitment(
					ctx,
					preConfirmation.Commitment,
					preConfirmation.Signature,
				)
				if err != nil {
					p.logger.Error("storing commitment", "error", err)
					return err
				}
				return w.WriteMsg(ctx, preConfirmation)
			}
		}
	} else {
		p.logger.Error("bidder does not have enough allowance", "ethAddress", ethAddress)
		return errors.New("bidder not allowed")
	}

	return nil
}
