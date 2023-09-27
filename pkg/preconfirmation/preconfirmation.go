package preconfirmation

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"

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
			{ // This is going to be a stream exclusively from the builder to the searcher
				Name:    "commitment",
				Handler: p.handleCommitment,
			},
		},
	}
}

// handlecommitment is meant to be used by the searcher exclusively to read the commitment value from the builder.
// They should verify the authenticity of the commitment
func (p *Preconfirmation) handleCommitment(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {
	r, _ := msgpack.NewReaderWriter[preconf.PreconfCommitment, preconf.PreconfCommitment](stream)
	commitment, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	// Process commitment as a searcher
	providerAddress, err := commitment.VerifyBuilderSignature()
	userAddress, err := commitment.VerifySearcherSignature()
	_ = providerAddress
	_ = userAddress

	// Check that user address is personal address
	// me == useraddress

	return nil
}

func (p *Preconfirmation) verifyBid(
	bid *PreConfBid,
) (common.Address, error) {

	ethAddress, err := bid.VerifySearcherSignature()
	if err != nil {
		return common.Address{}, err
	}

	return ethAddress, nil
}

// BroadcastBid sends the bid to the specified peer
func (p *Preconfirmation) BroadcastBid(ctx context.Context, peer p2p.Peer, bid *PreConfBid) error {
	stream, err := p.streamer.NewStream(ctx, peer, ProtocolName, ProtocolVersion, "bid")
	if err != nil {
		return err
	}

	_, w := msgpack.NewReaderWriter[PreConfBid, PreConfBid](stream)
	if err := w.WriteMsg(ctx, bid); err != nil {
		return err
	}

	return nil
}

// handlebid is the function that is called when a bid is received
// TODO(@ckartik):
// When you open a stream with a searcher - the other side of the stream has the handler
func (p *Preconfirmation) handleBid(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {
	// TODO(@ckartik): Change to reader only once availble
	r, _ := msgpack.NewReaderWriter[preconf.PreConfBid, preconf.PreConfBid](stream)
	bid, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	ethAddress, err := p.verifyBid(bid)
	if err != nil {
		return err
	}

	/* TODO(@ckartik): Determine if the bid is to be acted on - e.g constructed into a pre-confimration */
	searcherStream, err := p.streamer.NewStream(ctx, p2p.Peer{EthAddress: ethAddress, Type: p2p.PeerTypeSearcher}, ProtocolName, ProtocolVersion, "commitment")
	if err != nil {
		return err
	}

	commitment, err := bid.ConstructCommitment(p.signer)
	if err != nil {
		return err
	}

	_, w := msgpack.NewReaderWriter[preconf.PreconfCommitment, preconf.PreconfCommitment](searcherStream)
	w.WriteMsg(ctx, &commitment)
	// if recieved from searcher broadcast to builder peers
	// elseif recieved from builder, don't broadcast

	// Query Focused on only selecting builder peers
	builderPeers := p.topo.GetPeers(topology.Query{})
	for _, peer := range builderPeers {
		// Send bid to peers
		err := p.BroadcastBid(ctx, peer, bid)
		if err != nil {
			return err
		}
	}

	return nil
}
