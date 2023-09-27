package preconfirmation

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
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
	topo     topology.Topology
	streamer P2PService
	sem      *semaphore.Weighted
	quit     chan struct{}
}

func New(topo topology.Topology, streamer P2PService) *Preconfirmation {
	return &Preconfirmation{
		topo:     topo,
		streamer: streamer,
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
			// { // This is going to be a stream exclusively from the builder to the searcher
			// 	Name:    "commitment",
			// 	Handler: p.handleCommitment,
			// },
		},
	}
}

type UnsignedPreConfBid struct {
	TxnHash     string   `json:"txnHash"`
	Bid         *big.Int `json:"bid"`
	Blocknumber *big.Int `json:"blocknumber"`
	// UUID    string `json:"uuid"` // Assuming string representation for byte16
}

type PreConfBid struct { // Adds blocknumber for pre-conf bid - Will need to manage how to reciever acts on a bid / TTL is the blocknumber
	UnsignedPreConfBid

	BidHash   []byte `json:"bidhash"`
	Signature []byte `json:"signature"`
}

type PreconfCommitment struct {
	PreConfBid

	DataHash            []byte `json:"data_hash"`
	CommitmentSignature []byte `json:"commitment_signature"`
}

// golang interface
type IPreconfBid interface {
	GetTxnHash() string
	GetBidAmt() *big.Int
	VerifySearcherSignature() (common.Address, error)
	BidOriginator() (common.Address, *ecdsa.PublicKey, error)
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
func (p *Preconfirmation) handleBid(
	ctx context.Context,
	peer p2p.Peer,
	stream p2p.Stream,
) error {

	r, _ := msgpack.NewReaderWriter[PreConfBid, PreConfBid](stream)
	bid, err := r.ReadMsg(ctx)
	if err != nil {
		return err
	}

	ethAddress, err := p.verifyBid(bid)
	if err != nil {
		return err
	}

	/* TODO(@ckartik): Determine if the bid is to be acted on - e.g constructed into a pre-confimration */
	_ = ethAddress
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
