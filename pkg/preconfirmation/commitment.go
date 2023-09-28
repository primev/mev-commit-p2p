package preconfirmation

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/p2p/msgpack"
	"github.com/primevprotocol/mev-commit/pkg/structures/preconf"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"golang.org/x/sync/semaphore"
)

type CommitmentCallback struct {
	signer   preconf.Signer
	topo     topology.Topology
	streamer P2PService
	sem      *semaphore.Weighted
	quit     chan struct{}
}

func (cc *CommitmentCallback) Protocol() p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{ // This is going to be a stream exclusively from the builder to the searcher
				Name:    "commitment",
				Handler: cc.handleCommitment,
			},
		},
	}
}

func (cc *CommitmentCallback) ConstructAndSendCommitment(ctx context.Context, userAddress common.Address, bid *preconf.PreConfBid) error {
	taregetUserPeer := p2p.Peer{EthAddress: userAddress, Type: p2p.PeerTypeSearcher}

	// Create a new connection

	/* TODO(@ckartik): Determine if the bid is to be acted on - e.g constructed into a pre-confimration */
	searcherStream, err := cc.streamer.NewStream(ctx, taregetUserPeer, ProtocolName, ProtocolVersion, "commitment")
	if err != nil {
		return err
	}

	commitment, err := bid.ConstructCommitment(cc.signer)
	if err != nil {
		return err
	}

	_, w := msgpack.NewReaderWriter[preconf.PreconfCommitment, preconf.PreconfCommitment](searcherStream)
	err = w.WriteMsg(ctx, &commitment)

	return err
}

// handlecommitment is meant to be used by the searcher exclusively to read the commitment value from the builder.
// They should verify the authenticity of the commitment
func (cc *CommitmentCallback) handleCommitment(
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

	/* TODO(@ckartik): Determine if the bid is to be acted on - e.g constructed into a pre-confimration */
	userStream, err := cc.streamer.NewStream(ctx, peer, ProtocolName, ProtocolVersion, "commitment")
	if err != nil {
		return err
	}

	_, w := msgpack.NewReaderWriter[preconf.PreconfCommitment, preconf.PreconfCommitment](userStream)
	err = w.WriteMsg(ctx, commitment)

	return err
}
