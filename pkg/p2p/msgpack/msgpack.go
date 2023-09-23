package msgpack

import (
	"context"
	"fmt"

	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/vmihailenco/msgpack/v5"
)

const delimitedReaderMaxSize = 1024 * 1024

func NewReaderWriter[Req any, Resp any](s p2p.Stream) (Reader[Req], Writer[Resp]) {
	return newReader[Req](s), newWriter[Resp](s)
}

type Reader[Msg any] struct {
	p2p.Stream
}

type Writer[Msg any] struct {
	p2p.Stream
}

func newReader[Msg any](s p2p.Stream) Reader[Msg] {
	return Reader[Msg]{s}
}

func newWriter[Msg any](s p2p.Stream) Writer[Msg] {
	return Writer[Msg]{s}
}

func (r Reader[Msg]) ReadMsg(ctx context.Context) (*Msg, error) {
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
		msg := new(Msg)
		if err := msgpack.Unmarshal(res.msgBuf, msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal msg: %w", err)
		}
		return msg, nil
	}
}

func (w Writer[Msg]) WriteMsg(ctx context.Context, msg *Msg) error {
	msgBuf, err := msgpack.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed marshaling msg: %w", err)
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
