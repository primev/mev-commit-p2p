package libp2p

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
)

func newStream(libp2pstream network.Stream) p2p.Stream {
	return &stream{
		Stream: libp2pstream,
		reader: msgio.NewVarintReaderSize(libp2pstream, network.MessageSizeMax),
		writer: msgio.NewVarintWriter(libp2pstream),
	}
}

type stream struct {
	network.Stream
	reader msgio.Reader
	writer msgio.Writer
}

func (s *stream) ReadMsg() ([]byte, error) {
	return s.reader.ReadMsg()
}

func (s *stream) WriteMsg(msg []byte) error {
	return s.writer.WriteMsg(msg)
}
