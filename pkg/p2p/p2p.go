package p2p

import (
	"context"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

// PeerType is the type of a peer
type PeerType int

const (
	// PeerTypeBootnode is a boot node
	PeerTypeBootnode PeerType = iota
	// PeerTypeProvider is a provider node
	PeerTypeProvider
	// PeerTypeBidder is a bidder node
	PeerTypeBidder
	// PeerTypeRelay is a relay node
	PeerTypeRelay
)

func (pt PeerType) String() string {
	switch pt {
	case PeerTypeBootnode:
		return "bootnode"
	case PeerTypeProvider:
		return "provider"
	case PeerTypeBidder:
		return "bidder"
	case PeerTypeRelay:
		return "relay"
	default:
		return "unknown"
	}
}

func FromString(str string) PeerType {
	switch str {
	case "bootnode":
		return PeerTypeBootnode
	case "provider":
		return PeerTypeProvider
	case "bidder":
		return PeerTypeBidder
	case "relay":
		return PeerTypeRelay
	default:
		return -1
	}
}

var (
	ErrPeerNotFound = errors.New("peer not found")
	ErrNoAddresses  = errors.New("no addresses")
)

type Peer struct {
	EthAddress common.Address
	Type       PeerType
}

type PeerInfo struct {
	EthAddress common.Address
	Underlay   []byte
}

type Stream interface {
	ReadMsg() ([]byte, error)
	WriteMsg([]byte) error

	Reset() error
	io.Closer

	CloseWrite() error
	CloseRead() error
}

type Handler func(ctx context.Context, peer Peer, stream Stream) error

type StreamSpec struct {
	Name    string
	Handler Handler
}

type ProtocolSpec struct {
	Name        string
	Version     string
	StreamSpecs []StreamSpec
}

type Addressbook interface {
	GetPeerInfo(Peer) ([]byte, error)
}

type Streamer interface {
	NewStream(ctx context.Context, peer Peer, proto, version, stream string) (Stream, error)
}

type Service interface {
	AddProtocol(spec ProtocolSpec)
	Connect(ctx context.Context, info []byte) (Peer, error)
	Streamer
	Addressbook
	// Peers blocklisted by libp2p. Currently no external service needs the blocklist
	// so we don't expose it.
	BlockedPeers() []BlockedPeerInfo
	io.Closer
}

type Notifier interface {
	Connected(Peer)
	Disconnected(Peer)
}

type BlockedPeerInfo struct {
	Peer     common.Address
	Reason   string
	Duration string
}

func NewStreamName(protocol, version, stream string) string {
	return "/primev/" + protocol + "/" + version + "/" + stream
}
