// package client implements a simple gRPC client which is to be run by the builder
// in their environment to get a stream of bids that are being gossip'd in the
// mev-commit network. The builders can then decide to accept or reject the bid.
// This status is sent back to the mev-commit node to further take action on the
// network. The client can be improved by handling connection failures or using
// pool of connections etc.
package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	"google.golang.org/grpc"
)

type BuilderClient struct {
	conn         *grpc.ClientConn
	client       builderapiv1.BuilderClient
	logger       *slog.Logger
	senderC      chan *builderapiv1.BidResponse
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

	b := &BuilderClient{
		conn:         conn,
		client:       client,
		logger:       logger,
		senderC:      make(chan *builderapiv1.BidResponse),
		senderClosed: make(chan struct{}),
	}

	b.startSender()

	return b, nil
}

func (b *BuilderClient) Close() error {
	close(b.senderC)
	return b.conn.Close()
}

func (b *BuilderClient) startSender() error {
	fmt.Println("starting sender")

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
				err := stream.Send(resp)
				if err != nil {
					b.logger.Error("failed sending response", "error", err)
				}
			}
		}
	}()

	return nil
}

// ReceiveBids opens a new RPC connection with the mev-commit node to receive bids.
// Each call to this function opens a new connection and the bids are randomly
// assigned to one of the existing connections from mev-commit node. So if you run
// multiple listeners, they will get unique bids in a non-deterministic fashion.
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

// SendBidResponse can be used to send the status of the bid back to the mev-commit
// node. The builder can use his own logic to decide upon the bid and once he is
// ready to make a decision, this status has to be sent back to mev-commit to decide
// what to do with this bid. The sender is a single global worker which sends back
// the messages on grpc.
func (b *BuilderClient) SendBidResponse(
	ctx context.Context,
	bidResponse *builderapiv1.BidResponse,
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
