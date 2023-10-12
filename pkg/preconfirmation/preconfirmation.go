package preconfirmation

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/primevcrypto"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "1.0.0"
)

type Preconfirmation struct {
	signer    primevcrypto.Signer
	topo      Topology
	streamer  p2p.Streamer
	us        UserStore
	processer BidProcesser
	logger    *slog.Logger
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

type UserStore interface {
	CheckUserRegistred(*common.Address) bool
}

type BidProcesser interface {
	ProcessBid(context.Context, *primevcrypto.Bid) (chan builderapiv1.BidResponse_Status, error)
}

func New(
	topo Topology,
	streamer p2p.Streamer,
	signer primevcrypto.Signer,
	us UserStore,
	processor BidProcesser,
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

// SendBid is meant to be called by the searcher to construct and send bids to the builder.
// It takes the txnHash, the bid amount in wei and the maximum valid block number.
// It waits for preConfirmations from all builders and then returns.
// It returns an error if the bid is not valid.
func (p *Preconfirmation) SendBid(
	ctx context.Context,
	txnHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
) (chan *primevcrypto.PreConfirmation, error) {
	signedBid, err := p.signer.ConstructSignedBid(txnHash, bidAmt, blockNumber)
	if err != nil {
		p.logger.Error("constructing signed bid", "err", err, "txnHash", txnHash)
		return nil, err
	}

	builders := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeBuilder})
	if len(builders) == 0 {
		p.logger.Error("no builders available", "txnHash", txnHash)
		return nil, errors.New("no builders available")
	}

	// Create a new channel to receive preConfirmations
	preConfirmations := make(chan *primevcrypto.PreConfirmation, len(builders))

	wg := sync.WaitGroup{}
	for idx := range builders {
		wg.Add(1)
		go func(builder p2p.Peer) {
			defer wg.Done()

			logger := p.logger.With("builder", builder, "bid", txnHash)

			builderStream, err := p.streamer.NewStream(
				ctx,
				builder,
				ProtocolName,
				ProtocolVersion,
				"bid",
			)
			if err != nil {
				logger.Error("creating stream", "err", err)
				return
			}

			r, w := msgpack.NewReaderWriter[primevcrypto.PreConfirmation, primevcrypto.Bid](builderStream)
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

			// Process preConfirmation as a searcher
			_, err = p.signer.VerifyPreConfirmation(preConfirmation)
			if err != nil {
				logger.Error("verifying builder signature", "err", err)
				return
			}

			select {
			case preConfirmations <- preConfirmation:
			case <-ctx.Done():
				logger.Error("context cancelled", "err", ctx.Err())
				return
			}
		}(builders[idx])
	}

	go func() {
		wg.Wait()
		close(preConfirmations)
	}()

	return preConfirmations, nil
}

var ErrInvalidSearcherTypeForBid = errors.New("invalid searcher type for bid")

// handlebid is the function that is called when a bid is received
// It is meant to be used by the builder exclusively to read the bid value from the searcher.
func (p *Preconfirmation) handleBid(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {
	if peer.Type != p2p.PeerTypeSearcher {
		return ErrInvalidSearcherTypeForBid
	}

	// TODO(@ckartik): Change to reader only once availble
	r, w := msgpack.NewReaderWriter[primevcrypto.Bid, primevcrypto.PreConfirmation](stream)
	bid, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	ethAddress, err := p.signer.VerifyBid(bid)
	if err != nil {
		return err
	}

	if p.us.CheckUserRegistred(ethAddress) {
		statusC, err := p.processer.ProcessBid(ctx, bid)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case status := <-statusC:
			switch status {
			case builderapiv1.BidResponse_STATUS_REJECTED:
				return errors.New("bid rejected")
			case builderapiv1.BidResponse_STATUS_ACCEPTED:
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
