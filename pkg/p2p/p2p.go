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
	// PeerTypeBuilder is a builder node
	PeerTypeBuilder
	// PeerTypeSearcher is a searcher node
	PeerTypeSearcher
)

func (pt PeerType) String() string {
	switch pt {
	case PeerTypeBootnode:
		return "bootnode"
	case PeerTypeBuilder:
		return "builder"
	case PeerTypeSearcher:
		return "searcher"
	default:
		return "unknown"
	}
}

func FromString(str string) PeerType {
	switch str {
	case "bootnode":
		return PeerTypeBootnode
	case "builder":
		return PeerTypeBuilder
	case "searcher":
		return PeerTypeSearcher
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

type Stream interface {
	ReadMsg() ([]byte, error)
	WriteMsg([]byte) error

	Reset() error
	io.Closer
}

type StreamSpec struct {
	Name    string
	Handler func(ctx context.Context, peer Peer, stream Stream) error
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
