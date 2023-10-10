package client

import (
	"context"

	builderapiv1 "github.com/primevprotocol/mev-commit/gen/go/rpc/builderapi/v1"
	"google.golang.org/grpc"
)

type BuilderClient struct {
	conn   *grpc.ClientConn
	client builderapiv1.BuilderClient
}

func NewBuilderClient(serverAddr string) (*BuilderClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := builderapiv1.NewBuilderClient(conn)

	return &BuilderClient{
		conn:   conn,
		client: client,
	}, nil
}

func (b *BuilderClient) Close() {
	b.conn.Close()
}

func (b *BuilderClient) ReceiveBids() ([]*builderapiv1.Bid, error) {
	emptyMessage := &builderapiv1.EmptyMessage{}
	bidStream, err := b.client.ReceiveBids(context.Background(), emptyMessage)
	if err != nil {
		return nil, err
	}

	var bids []*builderapiv1.Bid
	for {
		bid, err := bidStream.Recv()
		if err != nil {
			return bids, err
		}

		var bidResponses []*builderapiv1.BidResponse
		bidResponse := &builderapiv1.BidResponse{
			BidHash: []byte(bid.GetBidHash()),
			Status:  builderapiv1.BidResponse_STATUS_ACCEPTED,
		}
		bidResponses = append(bidResponses, bidResponse)
		b.SendProcessedBids(bidResponses)
	}
}

func (b *BuilderClient) SendProcessedBids(bidResponses []*builderapiv1.BidResponse) error {
	stream, err := b.client.SendProcessedBids(context.Background())
	if err != nil {
		return err
	}

	for _, bidResponse := range bidResponses {
		if err := stream.Send(bidResponse); err != nil {
			return err
		}
	}

	//_, err = stream.CloseAndRecv()
	return err
}
