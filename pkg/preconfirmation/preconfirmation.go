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
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/signer/preconfsigner"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "1.0.0"
)

type Preconfirmation struct {
	signer    preconfsigner.Signer
	topo      Topology
	streamer  p2p.Streamer
	us        UserStore
	processer BidProcessor
	logger    *slog.Logger
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

type UserStore interface {
	CheckUserRegistred(*common.Address) bool
}

type BidProcessor interface {
	ProcessBid(context.Context, *preconfsigner.Bid) (chan providerapiv1.BidResponse_Status, error)
}

func New(
	topo Topology,
	streamer p2p.Streamer,
	signer preconfsigner.Signer,
	us UserStore,
	processor BidProcessor,
	logger *slog.Logger,
) *Preconfirmation {
	return &Preconfirmation{
		topo:      topo,
		streamer:  streamer,
		signer:    signer,
		us:        us,
		processer: processor,
		logger:    logger,
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

// SendBid is meant to be called by the user to construct and send bids to the provider.
// It takes the txHash, the bid amount in wei and the maximum valid block number.
// It waits for preConfirmations from all providers and then returns.
// It returns an error if the bid is not valid.
func (p *Preconfirmation) SendBid(
	ctx context.Context,
	txHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
) (chan *preconfsigner.PreConfirmation, error) {
	signedBid, err := p.signer.ConstructSignedBid(txHash, bidAmt, blockNumber)
	if err != nil {
		p.logger.Error("constructing signed bid", "err", err, "txHash", txHash)
		return nil, err
	}

	providers := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeProvider})
	if len(providers) == 0 {
		p.logger.Error("no providers available", "txHash", txHash)
		return nil, errors.New("no providers available")
	}

	// Create a new channel to receive preConfirmations
	preConfirmations := make(chan *preconfsigner.PreConfirmation, len(providers))

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
				logger.Error("creating stream", "err", err)
				return
			}

			r, w := msgpack.NewReaderWriter[preconfsigner.PreConfirmation, preconfsigner.Bid](providerStream)
			err = w.WriteMsg(ctx, signedBid)
			if err != nil {
				logger.Error("writing message", "err", err)
				return
			}

			preConfirmation, err := r.ReadMsg(ctx)
			if err != nil {
				logger.Error("reading message", "err", err)
				return
			}

			// Process preConfirmation as a user
			_, err = p.signer.VerifyPreConfirmation(preConfirmation)
			if err != nil {
				logger.Error("verifying provider signature", "err", err)
				return
			}

			select {
			case preConfirmations <- preConfirmation:
			case <-ctx.Done():
				logger.Error("context cancelled", "err", ctx.Err())
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

var ErrInvalidUserTypeForBid = errors.New("invalid user type for bid")

// handlebid is the function that is called when a bid is received
// It is meant to be used by the provider exclusively to read the bid value from the user.
func (p *Preconfirmation) handleBid(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {
	if peer.Type != p2p.PeerTypeUser {
		return ErrInvalidUserTypeForBid
	}

	// TODO(@ckartik): Change to reader only once availble
	r, w := msgpack.NewReaderWriter[preconfsigner.Bid, preconfsigner.PreConfirmation](stream)
	bid, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	ethAddress, err := p.signer.VerifyBid(bid)
	if err != nil {
		return err
	}

	if p.us.CheckUserRegistred(ethAddress) {
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
				preConfirmation, err := p.signer.ConstructPreConfirmation(bid)
				if err != nil {
					return err
				}
				return w.WriteMsg(ctx, preConfirmation)
			}
		}
	}

	return nil
}
