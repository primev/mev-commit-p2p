package client

import (
	"context"
	"errors"
	"log/slog"

	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	"google.golang.org/grpc"
)

type BuilderClient struct {
	conn         *grpc.ClientConn
	client       builderapiv1.BuilderClient
	logger       *slog.Logger
	senderC      chan builderapiv1.BidResponse
	senderClosed chan struct{}
}

func NewBuilderClient(
	serverAddr string,
	logger *slog.Logger,
) (*BuilderClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := builderapiv1.NewBuilderClient(conn)

	return &BuilderClient{
		conn:         conn,
		client:       client,
		logger:       logger,
		senderC:      make(chan builderapiv1.BidResponse),
		senderClosed: make(chan struct{}),
	}, nil
}

func (b *BuilderClient) Close() error {
	close(b.senderC)
	return b.conn.Close()
}

func (b *BuilderClient) startSender() error {
	stream, err := b.client.SendProcessedBids(context.Background())
	if err != nil {
		return err
	}

	go func() {
		defer close(b.senderClosed)
		for {
			select {
			case <-stream.Context().Done():
				b.logger.Warn("closing client conn")
				return
			case resp, more := <-b.senderC:
				if !more {
					b.logger.Warn("closed sender chan")
					return
				}
				err := stream.Send(&resp)
				if err != nil {
					b.logger.Error("failed sending response", "error", err)
				}
			}
		}
	}()

	return nil
}

func (b *BuilderClient) ReceiveBids() (chan *builderapiv1.Bid, error) {
	emptyMessage := &builderapiv1.EmptyMessage{}
	bidStream, err := b.client.ReceiveBids(context.Background(), emptyMessage)
	if err != nil {
		return nil, err
	}

	bidC := make(chan *builderapiv1.Bid)
	go func() {
		defer close(bidC)
		for {
			bid, err := bidStream.Recv()
			if err != nil {
				b.logger.Error("failed receiving bid", "error", err)
				return
			}
			select {
			case <-bidStream.Context().Done():
			case bidC <- bid:
			}
		}
	}()

	return bidC, nil
}

func (b *BuilderClient) SendBidResponse(
	ctx context.Context,
	bidResponse builderapiv1.BidResponse,
) error {

	select {
	case <-b.senderClosed:
		return errors.New("sender closed")
	default:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case b.senderC <- bidResponse:
		return nil
	}
}
