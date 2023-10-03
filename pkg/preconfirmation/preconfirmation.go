package preconfirmation

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"log/slog"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/preconf"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "1.0.0"
)

type Preconfirmation struct {
	signer   preconf.Signer
	topo     Topology
	streamer p2p.Streamer
	us       UserStore
	logger   *slog.Logger
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

type UserStore interface {
	CheckUserRegistred(*common.Address) bool
}

func New(
	topo Topology,
	streamer p2p.Streamer,
	key *ecdsa.PrivateKey,
	us UserStore,
	logger *slog.Logger,
) *Preconfirmation {
	return &Preconfirmation{
		topo:     topo,
		streamer: streamer,
		signer:   preconf.PrivateKeySigner{PrivKey: key},
		us:       us,
		logger:   logger,
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
// It waits for commitments from all builders and then returns.
// It returns an error if the bid is not valid.
func (p *Preconfirmation) SendBid(
	ctx context.Context,
	txnHash string,
	bidAmt *big.Int,
	blockNumber *big.Int,
) (chan *preconf.PreconfCommitment, error) {
	signedBid, err := preconf.ConstructSignedBid(bidAmt, txnHash, blockNumber, p.signer)
	if err != nil {
		p.logger.Error("constructing signed bid", "err", err, "txnHash", txnHash)
		return nil, err
	}

	builders := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeBuilder})
	if len(builders) == 0 {
		p.logger.Error("no builders available", "txnHash", txnHash)
		return nil, errors.New("no builders available")
	}

	// Create a new channel to receive commitments
	commitments := make(chan *preconf.PreconfCommitment, len(builders))

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

			r, w := msgpack.NewReaderWriter[preconf.PreconfCommitment, preconf.PreConfBid](builderStream)
			err = w.WriteMsg(ctx, signedBid)
			if err != nil {
				logger.Error("writing message", "err", err)
				return
			}

			commitment, err := r.ReadMsg(ctx)
			if err != nil {
				logger.Error("reading message", "err", err)
				return
			}

			// Process commitment as a searcher
			_, err = commitment.VerifyBuilderSignature()
			if err != nil {
				logger.Error("verifying builder signature", "err", err)
				return
			}

			_, err = commitment.VerifySearcherSignature()
			if err != nil {
				logger.Error("verifying searcher signature", "err", err)
				return
			}

			select {
			case commitments <- commitment:
			case <-ctx.Done():
				logger.Error("context cancelled", "err", ctx.Err())
				return
			}
		}(builders[idx])
	}

	go func() {
		wg.Wait()
		close(commitments)
	}()

	return commitments, nil
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
	r, w := msgpack.NewReaderWriter[preconf.PreConfBid, preconf.PreconfCommitment](stream)
	bid, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	ethAddress, err := bid.VerifySearcherSignature()
	if err != nil {
		return err
	}

	if p.us.CheckUserRegistred(ethAddress) {
		// More conditional Logic to determine signing of bid
		commitment, err := preconf.ConstructCommitment(*bid, p.signer)
		if err != nil {
			return err
		}

		return w.WriteMsg(ctx, commitment)
	}

	return nil
}
