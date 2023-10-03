package protobuf

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/p2p/host/autonat/pb"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"google.golang.org/protobuf/proto"
)

type Encoder interface {
	ReadMsg(context.Context, *pb.Message) error
	WriteMsg(context.Context, *pb.Message) error
}

type protobuf struct {
	p2p.Stream
}

func NewReaderWriter(s p2p.Stream) Encoder {
	return &protobuf{s}
}

func (p *protobuf) ReadMsg(ctx context.Context, msg *pb.Message) error {
	type result struct {
		msgBuf []byte
		err    error
	}

	resultC := make(chan result, 1)
	go func() {
		msgBuf, err := p.Stream.ReadMsg()
		resultC <- result{msgBuf: msgBuf, err: err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-resultC:
		if res.err != nil {
			return fmt.Errorf("failed to read msg: %w", res.err)
		}

		if err := proto.Unmarshal(res.msgBuf, msg); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		return nil
	}
}

func (p *protobuf) WriteMsg(ctx context.Context, msg *pb.Message) error {
	msgBuf, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed marshaling message: %w", err)
	}

	errC := make(chan error, 1)
	go func() {
		errC <- p.Stream.WriteMsg(msgBuf)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
