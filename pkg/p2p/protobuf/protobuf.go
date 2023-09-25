package protobuf

import (
	"context"
	"fmt"

	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"google.golang.org/protobuf/proto"
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

func (r Reader) ReadMsg(ctx context.Context, msg proto.Message) error {
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

func (w Writer) WriteMsg(ctx context.Context, msg proto.Message) error {
	msgBuf, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed marshaling message: %w", err)
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
