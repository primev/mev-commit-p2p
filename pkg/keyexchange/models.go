package keyexchange

import (
	"errors"
	"log/slog"

	"github.com/primevprotocol/mev-commit/pkg/keykeeper"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/signer"
	"github.com/primevprotocol/mev-commit/pkg/topology"
)

// Protocol constants.
const (
	ProtocolName        = "keyexchange"
	ProtocolHandlerName = "timestampMessage"
	ProtocolVersion     = "1.0.0"
)

// Error declarations.
var (
	ErrSignatureVerificationFailed = errors.New("signature verification failed")
	ErrObservedAddressMismatch     = errors.New("observed address mismatch")
	ErrInvalidBidderTypeForMessage = errors.New("invalid bidder type for message")
	ErrNoProvidersAvailable        = errors.New("no providers available")
)

// KeyExchange manages the key exchange process.
type KeyExchange struct {
	keyKeeper      keykeeper.KeyKeeper
	topo           Topology
	streamer       p2p.Streamer
	signer         signer.Signer
	logger         *slog.Logger
}

// Topology interface to get peers.
type Topology interface {
	GetPeers(topology.Query) []p2p.Peer
}
