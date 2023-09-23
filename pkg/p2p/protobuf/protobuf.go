package protobuf

import (
	"context"
	"fmt"

	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const delimitedReaderMaxSize = 1024 * 1024

func NewReaderWriter(s p2p.Stream) (Reader, Writer) {
	return newReader(s), newWriter(s)
}

type Reader struct {
	p2p.Stream
}

type Writer struct {
	p2p.Stream
}

func newReader(s p2p.Stream) Reader {
	return Reader{s}
}

func newWriter(s p2p.Stream) Writer {
	return Writer{s}
}

func (r Reader) ReadMsg(ctx context.Context) (*anypb.Any, error) {
	type result struct {
		msgBuf []byte
		err    error
	}

	resultC := make(chan result, 1)
	go func() {
		msgBuf, err := r.Stream.ReadMsg()
		resultC <- result{msgBuf: msgBuf, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultC:
		if res.err != nil {
			return nil, fmt.Errorf("failed to read msg: %w", res.err)
		}

		msg := &anypb.Any{}
		if err := msg.Unmarshal(res.msgBuf); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Any message: %w", err)
		}

		return msg, nil
	}
}

func (w Writer) WriteMsg(ctx context.Context, msg proto.Message) error {
	anyMessage, err := anypb.New(msg)
	if err != nil {
		return fmt.Errorf("failed to create Any message: %w", err)
	}

	msgBuf, err := anyMessage.Marshal()
	if err != nil {
		return fmt.Errorf("failed marshaling Any message: %w", err)
	}

	errC := make(chan error, 1)
	go func() {
		errC <- w.Stream.WriteMsg(msgBuf)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
