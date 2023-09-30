package preconfirmation

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/structures/preconf"
	"github.com/primevprotocol/mev-commit/pkg/topology"
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
	topo     Topology
	streamer P2PService
	cs       CommitmentsStore
	us       UserStore
}

type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}

func New(topo Topology, streamer P2PService, key *ecdsa.PrivateKey) *Preconfirmation {
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

// BidHash -> map of preconfs
// Key: BidHash
// Value: List of preconfs
// var commitments map[string][]preconf.PreconfCommitment
type CommitmentsStore interface {
	GetCommitments(bidHash []byte) ([]preconf.PreconfCommitment, error)
	AddCommitment(bidHash []byte, commitment *preconf.PreconfCommitment) error
}

type UserStore interface {
	CheckUserRegistred(common.Address) bool
}

/*

	TxnHash     string   `json:"txnHash"`
	Bid         *big.Int `json:"bid"`
	Blocknumber *big.Int `json:"blocknumber"`
*/
// SendBid is meant to be called by the searcher to construct and send bids to the builder
func (p *Preconfirmation) SendBid(ctx context.Context, txnHash string, bidamt *big.Int, blockNumber *big.Int) error {
	signedBid, err := preconf.ConstructSignedBid(bidamt, txnHash, blockNumber, p.signer)
	if err != nil {
		return err
	}

	builders := p.topo.GetPeers(topology.Query{Type: p2p.PeerTypeBuilder})

	// TODO(@ckartik): Push into a channel and process in parallel
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
		if err != nil {
			return err
		}

		// Process commitment as a searcher
		providerAddress, err := commitment.VerifyBuilderSignature()
		if err != nil {
			return err
		}
		userAddress, err := commitment.VerifySearcherSignature()
		if err != nil {
			return err
		}

		_ = providerAddress
		_ = userAddress

		// Verify the bid details correspond.
		err = p.cs.AddCommitment(signedBid.BidHash, commitment)
		if err != nil {
			return err
		}
	}

	return nil
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

		return w.WriteMsg(ctx, &commitment)
	}

	return nil
}
