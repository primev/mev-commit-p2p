package libp2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio"
	streammsgv1 "github.com/primevprotocol/mev-commit/gen/go/streammsg/v1"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func newStream(libp2pstream network.Stream, hdrs, respHdrs p2p.Header) p2p.Stream {
	return &stream{
		Stream:   libp2pstream,
		reader:   msgio.NewVarintReaderSize(libp2pstream, network.MessageSizeMax),
		writer:   msgio.NewVarintWriter(libp2pstream),
		hdrs:     hdrs,
		respHdrs: respHdrs,
	}
}

type stream struct {
	network.Stream
	reader   msgio.Reader
	writer   msgio.Writer
	hdrs     p2p.Header
	respHdrs p2p.Header
}

type result struct {
	buf []byte
	err error
}

func (s *stream) ReadMsg(ctx context.Context, m proto.Message) error {
	ch := make(chan result, 1)
	go func() {
		sMsgBuf, err := s.reader.ReadMsg()
		ch <- result{sMsgBuf, err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return fmt.Errorf("failed to read message: %w", res.err)
		}

		sMsg := new(streammsgv1.StreamMsg)
		err := proto.Unmarshal(res.buf, sMsg)
		if err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		if sMsg.GetError() != nil {
			return status.FromProto(sMsg.GetError()).Err()
		}

		if sMsg.GetData() == nil {
			return fmt.Errorf("message has no data")
		}

		return proto.Unmarshal(sMsg.GetData(), m)
	}
}

func (s *stream) WriteMsg(ctx context.Context, m proto.Message) error {
	msg, err := proto.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	sMsg := &streammsgv1.StreamMsg{
		Body: &streammsgv1.StreamMsg_Data{
			Data: msg,
		},
	}

	sMsgBuf, err := proto.Marshal(sMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	errC := make(chan error, 1)

	go func() {
		errC <- s.writer.WriteMsg(sMsgBuf)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

type metadataStream struct {
	network.Stream
	reader msgio.Reader
	writer msgio.Writer
}

func newMetadataStream(libp2pstream network.Stream) p2p.MetadataStream {
	return &metadataStream{
		Stream: libp2pstream,
		reader: msgio.NewVarintReaderSize(libp2pstream, network.MessageSizeMax),
		writer: msgio.NewVarintWriter(libp2pstream),
	}
}

func (s *metadataStream) ReadHeader(ctx context.Context) (p2p.Header, error) {
	ch := make(chan result, 1)
	go func() {
		sMsgBuf, err := s.reader.ReadMsg()
		ch <- result{sMsgBuf, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("failed to read header: %w", res.err)
		}
		header := new(streammsgv1.Header)
		err := proto.Unmarshal(res.buf, header)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal header: %w", err)
		}
		return header.Header, nil
	}
}

func (s *metadataStream) WriteHeader(ctx context.Context, hdr p2p.Header) error {
	sMsg := &streammsgv1.Header{
		Header: hdr,
	}

	sMsgBuf, err := proto.Marshal(sMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal header: %w", err)
	}

	errC := make(chan error, 1)
	go func() {
		errC <- s.writer.WriteMsg(sMsgBuf)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

func (s *metadataStream) WriteError(ctx context.Context, st *status.Status) error {
	sMsg := &streammsgv1.StreamMsg{
		Body: &streammsgv1.StreamMsg_Error{
			Error: st.Proto(),
		},
	}

	buf, err := proto.Marshal(sMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal error message: %w", err)
	}

	errC := make(chan error, 1)
	go func() {
		errC <- s.writer.WriteMsg(buf)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
