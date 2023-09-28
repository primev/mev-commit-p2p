package preconfirmation

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"

	"github.com/primevprotocol/mev-commit/pkg/structures/preconf"

	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"golang.org/x/sync/semaphore"
)

const (
	ProtocolName    = "preconfirmation"
	ProtocolVersion = "1.0.0"
)

type P2PService interface {
	p2p.Streamer
	p2p.Addressbook
	Connect(context.Context, []byte) (p2p.Peer, error)
}

type Preconfirmation struct {
	signer   preconf.Signer
	topo     topology.Topology
	streamer P2PService
	sem      *semaphore.Weighted
	quit     chan struct{}
}

func New(topo topology.Topology, streamer P2PService, key *ecdsa.PrivateKey) *Preconfirmation {
	return &Preconfirmation{
		topo:     topo,
		streamer: streamer,
		signer:   preconf.PrivateKeySigner{PrivKey: key},
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

// Local store of bids out
var bidsOut = make(map[common.Address]preconf.PreConfBid)

func (p *Preconfirmation) SendBid(ctx context.Context, bid preconf.UnsignedPreConfBid) error {
	signedBid, err := preconf.ConvertIntoSignedBid(bid, p.signer)
	if err != nil {
		return err
	}

	builders := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeBuilder})
	for _, builder := range builders {
		// Create a new connection
		builderStream, err := p.streamer.NewStream(ctx, builder, ProtocolName, ProtocolVersion, "bid")
		if err != nil {
			return err
		}

		r, w := msgpack.NewReaderWriter[preconf.PreconfCommitment, preconf.PreConfBid](builderStream)
		err = w.WriteMsg(ctx, &signedBid)
		if err != nil {
			return err
		}

		commitment, err := r.ReadMsg(ctx)

		// Process commitment as a searcher
		providerAddress, err := commitment.VerifyBuilderSignature()
		userAddress, err := commitment.VerifySearcherSignature()
		_ = providerAddress
		_ = userAddress

		// Verify the bid details correspond.
	}

	return nil
}

func (p *Preconfirmation) verifyBid(
	bid *preconf.PreConfBid,
) (common.Address, error) {

	ethAddress, err := bid.VerifySearcherSignature()
	if err != nil {
		return common.Address{}, err
	}

	return ethAddress, nil
}

var ErrInvalidSearcherTypeForBid = errors.New("invalid searcher type for bid")

// handlebid is the function that is called when a bid is received
// TODO(@ckartik):
// When you open a stream with a searcher - the other side of the stream has the handler
// handlebid could have a bid sent by a searcher node or a builder node.
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

	ethAddress, err := p.verifyBid(bid)
	if err != nil {
		return err
	}

	if peer.EthAddress != ethAddress {
		return errors.New("eth address does not match")
	}

	commitment, err := bid.ConstructCommitment(p.signer)
	if err != nil {
		return err
	}

	return w.WriteMsg(ctx, &commitment)
}
