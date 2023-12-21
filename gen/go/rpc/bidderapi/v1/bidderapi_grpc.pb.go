// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: rpc/bidderapi/v1/bidderapi.proto

package bidderapiv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Bidder_SendBid_FullMethodName = "/rpc.bidderapi.v1.Bidder/SendBid"
)

// BidderClient is the client API for Bidder service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BidderClient interface {
	// SendBid
	//
	// Send a bid to the bidder mev-commit node.
	SendBid(ctx context.Context, in *Bid, opts ...grpc.CallOption) (Bidder_SendBidClient, error)
}

type bidderClient struct {
	cc grpc.ClientConnInterface
}

func NewBidderClient(cc grpc.ClientConnInterface) BidderClient {
	return &bidderClient{cc}
}

func (c *bidderClient) SendBid(ctx context.Context, in *Bid, opts ...grpc.CallOption) (Bidder_SendBidClient, error) {
	stream, err := c.cc.NewStream(ctx, &Bidder_ServiceDesc.Streams[0], Bidder_SendBid_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &bidderSendBidClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Bidder_SendBidClient interface {
	Recv() (*Commitment, error)
	grpc.ClientStream
}

type bidderSendBidClient struct {
	grpc.ClientStream
}

func (x *bidderSendBidClient) Recv() (*Commitment, error) {
	m := new(Commitment)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BidderServer is the server API for Bidder service.
// All implementations must embed UnimplementedBidderServer
// for forward compatibility
type BidderServer interface {
	// SendBid
	//
	// Send a bid to the bidder mev-commit node.
	SendBid(*Bid, Bidder_SendBidServer) error
	mustEmbedUnimplementedBidderServer()
}

// UnimplementedBidderServer must be embedded to have forward compatible implementations.
type UnimplementedBidderServer struct {
}

func (UnimplementedBidderServer) SendBid(*Bid, Bidder_SendBidServer) error {
	return status.Errorf(codes.Unimplemented, "method SendBid not implemented")
}
func (UnimplementedBidderServer) mustEmbedUnimplementedBidderServer() {}

// UnsafeBidderServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BidderServer will
// result in compilation errors.
type UnsafeBidderServer interface {
	mustEmbedUnimplementedBidderServer()
}

func RegisterBidderServer(s grpc.ServiceRegistrar, srv BidderServer) {
	s.RegisterService(&Bidder_ServiceDesc, srv)
}

func _Bidder_SendBid_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Bid)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BidderServer).SendBid(m, &bidderSendBidServer{stream})
}

type Bidder_SendBidServer interface {
	Send(*Commitment) error
	grpc.ServerStream
}

type bidderSendBidServer struct {
	grpc.ServerStream
}

func (x *bidderSendBidServer) Send(m *Commitment) error {
	return x.ServerStream.SendMsg(m)
}

// Bidder_ServiceDesc is the grpc.ServiceDesc for Bidder service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Bidder_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rpc.bidderapi.v1.Bidder",
	HandlerType: (*BidderServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendBid",
			Handler:       _Bidder_SendBid_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "rpc/bidderapi/v1/bidderapi.proto",
}
