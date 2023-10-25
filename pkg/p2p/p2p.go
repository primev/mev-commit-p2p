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
	// PeerTypeUser is a user node
	PeerTypeUser
)

func (pt PeerType) String() string {
	switch pt {
	case PeerTypeBootnode:
		return "bootnode"
	case PeerTypeProvider:
		return "provider"
	case PeerTypeUser:
		return "user"
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
	case "user":
		return PeerTypeUser
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
	io.Closer
}

type Notifier interface {
	Connected(Peer)
	Disconnected(Peer)
}

func NewStreamName(protocol, version, stream string) string {
	return "/primev/" + protocol + "/" + version + "/" + stream
}
