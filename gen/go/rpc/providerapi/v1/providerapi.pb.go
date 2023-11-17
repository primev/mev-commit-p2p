// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: rpc/providerapi/v1/providerapi.proto

package providerapiv1

import (
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type BidResponse_Status int32

const (
	BidResponse_STATUS_UNSPECIFIED BidResponse_Status = 0
	BidResponse_STATUS_ACCEPTED    BidResponse_Status = 1
	BidResponse_STATUS_REJECTED    BidResponse_Status = 2
)

// Enum value maps for BidResponse_Status.
var (
	BidResponse_Status_name = map[int32]string{
		0: "STATUS_UNSPECIFIED",
		1: "STATUS_ACCEPTED",
		2: "STATUS_REJECTED",
	}
	BidResponse_Status_value = map[string]int32{
		"STATUS_UNSPECIFIED": 0,
		"STATUS_ACCEPTED":    1,
		"STATUS_REJECTED":    2,
	}
)

func (x BidResponse_Status) Enum() *BidResponse_Status {
	p := new(BidResponse_Status)
	*p = x
	return p
}

func (x BidResponse_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (BidResponse_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_rpc_providerapi_v1_providerapi_proto_enumTypes[0].Descriptor()
}

func (BidResponse_Status) Type() protoreflect.EnumType {
	return &file_rpc_providerapi_v1_providerapi_proto_enumTypes[0]
}

func (x BidResponse_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use BidResponse_Status.Descriptor instead.
func (BidResponse_Status) EnumDescriptor() ([]byte, []int) {
	return file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP(), []int{4, 0}
}

type StakeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Amount string `protobuf:"bytes,1,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (x *StakeRequest) Reset() {
	*x = StakeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StakeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StakeRequest) ProtoMessage() {}

func (x *StakeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StakeRequest.ProtoReflect.Descriptor instead.
func (*StakeRequest) Descriptor() ([]byte, []int) {
	return file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP(), []int{0}
}

func (x *StakeRequest) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

type StakeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Amount string `protobuf:"bytes,1,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (x *StakeResponse) Reset() {
	*x = StakeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StakeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StakeResponse) ProtoMessage() {}

func (x *StakeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StakeResponse.ProtoReflect.Descriptor instead.
func (*StakeResponse) Descriptor() ([]byte, []int) {
	return file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP(), []int{1}
}

func (x *StakeResponse) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

type EmptyMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *EmptyMessage) Reset() {
	*x = EmptyMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EmptyMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmptyMessage) ProtoMessage() {}

func (x *EmptyMessage) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EmptyMessage.ProtoReflect.Descriptor instead.
func (*EmptyMessage) Descriptor() ([]byte, []int) {
	return file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP(), []int{2}
}

type Bid struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TxHash      string `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	BidAmount   int64  `protobuf:"varint,2,opt,name=bid_amount,json=bidAmount,proto3" json:"bid_amount,omitempty"`
	BlockNumber int64  `protobuf:"varint,3,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	BidDigest   []byte `protobuf:"bytes,4,opt,name=bid_digest,json=bidDigest,proto3" json:"bid_digest,omitempty"`
}

func (x *Bid) Reset() {
	*x = Bid{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Bid) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bid) ProtoMessage() {}

func (x *Bid) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bid.ProtoReflect.Descriptor instead.
func (*Bid) Descriptor() ([]byte, []int) {
	return file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP(), []int{3}
}

func (x *Bid) GetTxHash() string {
	if x != nil {
		return x.TxHash
	}
	return ""
}

func (x *Bid) GetBidAmount() int64 {
	if x != nil {
		return x.BidAmount
	}
	return 0
}

func (x *Bid) GetBlockNumber() int64 {
	if x != nil {
		return x.BlockNumber
	}
	return 0
}

func (x *Bid) GetBidDigest() []byte {
	if x != nil {
		return x.BidDigest
	}
	return nil
}

type BidResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BidDigest []byte             `protobuf:"bytes,1,opt,name=bid_digest,json=bidDigest,proto3" json:"bid_digest,omitempty"`
	Status    BidResponse_Status `protobuf:"varint,2,opt,name=status,proto3,enum=rpc.providerapi.v1.BidResponse_Status" json:"status,omitempty"`
}

func (x *BidResponse) Reset() {
	*x = BidResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BidResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BidResponse) ProtoMessage() {}

func (x *BidResponse) ProtoReflect() protoreflect.Message {
	mi := &file_rpc_providerapi_v1_providerapi_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BidResponse.ProtoReflect.Descriptor instead.
func (*BidResponse) Descriptor() ([]byte, []int) {
	return file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP(), []int{4}
}

func (x *BidResponse) GetBidDigest() []byte {
	if x != nil {
		return x.BidDigest
	}
	return nil
}

func (x *BidResponse) GetStatus() BidResponse_Status {
	if x != nil {
		return x.Status
	}
	return BidResponse_STATUS_UNSPECIFIED
}

var File_rpc_providerapi_v1_providerapi_proto protoreflect.FileDescriptor

var file_rpc_providerapi_v1_providerapi_proto_rawDesc = []byte{
	0x0a, 0x24, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70,
	0x69, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32,
	0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x92, 0x01, 0x0a, 0x0c, 0x53, 0x74, 0x61,
	0x6b, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0x3a, 0x6a, 0x92, 0x41, 0x67, 0x0a, 0x41, 0x2a, 0x0d, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x20,
	0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x32, 0x28, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x20, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x20, 0x69, 0x6e, 0x20, 0x74, 0x68, 0x65, 0x20, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x20, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0xd2, 0x01, 0x05, 0x73, 0x74, 0x61, 0x6b, 0x65, 0x32, 0x22, 0x7b, 0x22, 0x61, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x22, 0x3a, 0x20, 0x22, 0x31, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
	0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x22, 0x20, 0x7d, 0x22, 0x9c, 0x01,
	0x0a, 0x0d, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x3a, 0x73, 0x92, 0x41, 0x70, 0x0a, 0x4a, 0x2a, 0x0e,
	0x53, 0x74, 0x61, 0x6b, 0x65, 0x20, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x38,
	0x47, 0x65, 0x74, 0x20, 0x73, 0x74, 0x61, 0x6b, 0x65, 0x64, 0x20, 0x61, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x20, 0x69,
	0x6e, 0x20, 0x74, 0x68, 0x65, 0x20, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x20, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x32, 0x22, 0x7b, 0x22, 0x61, 0x6d, 0x6f, 0x75,
	0x6e, 0x74, 0x22, 0x3a, 0x20, 0x22, 0x31, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
	0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x22, 0x20, 0x7d, 0x22, 0x0e, 0x0a, 0x0c,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0xb7, 0x06, 0x0a,
	0x03, 0x42, 0x69, 0x64, 0x12, 0x8f, 0x01, 0x0a, 0x07, 0x74, 0x78, 0x5f, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x76, 0x92, 0x41, 0x73, 0x32, 0x5f, 0x48, 0x65, 0x78,
	0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x20, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67,
	0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x68, 0x61, 0x73, 0x68, 0x20, 0x6f, 0x66, 0x20,
	0x74, 0x68, 0x65, 0x20, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x20,
	0x74, 0x68, 0x61, 0x74, 0x20, 0x74, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72, 0x20, 0x77, 0x61,
	0x6e, 0x74, 0x73, 0x20, 0x74, 0x6f, 0x20, 0x69, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x20, 0x69,
	0x6e, 0x20, 0x74, 0x68, 0x65, 0x20, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x2e, 0x8a, 0x01, 0x0f, 0x5b,
	0x61, 0x2d, 0x66, 0x41, 0x2d, 0x46, 0x30, 0x2d, 0x39, 0x5d, 0x7b, 0x34, 0x30, 0x7d, 0x52, 0x06,
	0x74, 0x78, 0x48, 0x61, 0x73, 0x68, 0x12, 0x8d, 0x01, 0x0a, 0x0a, 0x62, 0x69, 0x64, 0x5f, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x42, 0x6e, 0x92, 0x41, 0x6b,
	0x32, 0x69, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x45, 0x54, 0x48, 0x20,
	0x74, 0x68, 0x61, 0x74, 0x20, 0x74, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72, 0x20, 0x69, 0x73,
	0x20, 0x77, 0x69, 0x6c, 0x6c, 0x69, 0x6e, 0x67, 0x20, 0x74, 0x6f, 0x20, 0x70, 0x61, 0x79, 0x20,
	0x74, 0x6f, 0x20, 0x74, 0x68, 0x65, 0x20, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x20,
	0x66, 0x6f, 0x72, 0x20, 0x69, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x69, 0x6e, 0x67, 0x20, 0x74, 0x68,
	0x65, 0x20, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x69, 0x6e,
	0x20, 0x74, 0x68, 0x65, 0x20, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x2e, 0x52, 0x09, 0x62, 0x69, 0x64,
	0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x6b, 0x0a, 0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f,
	0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x42, 0x48, 0x92, 0x41,
	0x45, 0x32, 0x43, 0x4d, 0x61, 0x78, 0x20, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x20, 0x6e, 0x75, 0x6d,
	0x62, 0x65, 0x72, 0x20, 0x74, 0x68, 0x61, 0x74, 0x20, 0x74, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65,
	0x72, 0x20, 0x77, 0x61, 0x6e, 0x74, 0x73, 0x20, 0x74, 0x6f, 0x20, 0x69, 0x6e, 0x63, 0x6c, 0x75,
	0x64, 0x65, 0x20, 0x74, 0x68, 0x65, 0x20, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x20, 0x69, 0x6e, 0x2e, 0x52, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e, 0x75, 0x6d,
	0x62, 0x65, 0x72, 0x12, 0x51, 0x0a, 0x0a, 0x62, 0x69, 0x64, 0x5f, 0x64, 0x69, 0x67, 0x65, 0x73,
	0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x42, 0x32, 0x92, 0x41, 0x2f, 0x32, 0x2d, 0x44, 0x69,
	0x67, 0x65, 0x73, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x62, 0x69, 0x64, 0x20,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x20, 0x73, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x20, 0x62,
	0x79, 0x20, 0x74, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x52, 0x09, 0x62, 0x69, 0x64,
	0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0x3a, 0xcd, 0x02, 0x92, 0x41, 0xc9, 0x02, 0x0a, 0x6c, 0x2a,
	0x0b, 0x42, 0x69, 0x64, 0x20, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x2e, 0x53, 0x69,
	0x67, 0x6e, 0x65, 0x64, 0x20, 0x62, 0x69, 0x64, 0x20, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x20, 0x66, 0x72, 0x6f, 0x6d, 0x20, 0x75, 0x73, 0x65, 0x72, 0x73, 0x20, 0x74, 0x6f, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2e, 0xd2, 0x01, 0x06, 0x74,
	0x78, 0x48, 0x61, 0x73, 0x68, 0xd2, 0x01, 0x09, 0x62, 0x69, 0x64, 0x41, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0xd2, 0x01, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0xd2,
	0x01, 0x09, 0x62, 0x69, 0x64, 0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0x32, 0xd8, 0x01, 0x7b, 0x22,
	0x74, 0x78, 0x48, 0x61, 0x73, 0x68, 0x22, 0x3a, 0x20, 0x22, 0x39, 0x31, 0x61, 0x38, 0x39, 0x42,
	0x36, 0x33, 0x33, 0x31, 0x39, 0x34, 0x63, 0x30, 0x44, 0x38, 0x36, 0x43, 0x35, 0x33, 0x39, 0x41,
	0x31, 0x41, 0x35, 0x42, 0x31, 0x34, 0x44, 0x43, 0x43, 0x61, 0x63, 0x66, 0x44, 0x34, 0x37, 0x30,
	0x39, 0x34, 0x22, 0x2c, 0x20, 0x22, 0x62, 0x69, 0x64, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x22,
	0x3a, 0x20, 0x31, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
	0x30, 0x30, 0x30, 0x30, 0x30, 0x2c, 0x20, 0x22, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e, 0x75, 0x6d,
	0x62, 0x65, 0x72, 0x22, 0x3a, 0x20, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x2c, 0x20, 0x22, 0x62,
	0x69, 0x64, 0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0x22, 0x3a, 0x20, 0x22, 0x39, 0x64, 0x4a, 0x69,
	0x6e, 0x77, 0x4c, 0x2b, 0x46, 0x5a, 0x36, 0x42, 0x31, 0x78, 0x73, 0x49, 0x51, 0x51, 0x6f, 0x38,
	0x74, 0x38, 0x42, 0x30, 0x5a, 0x58, 0x4a, 0x75, 0x62, 0x4a, 0x77, 0x59, 0x38, 0x36, 0x6c, 0x2f,
	0x59, 0x75, 0x37, 0x79, 0x41, 0x48, 0x31, 0x35, 0x39, 0x51, 0x72, 0x50, 0x48, 0x55, 0x30, 0x71,
	0x6a, 0x32, 0x50, 0x2b, 0x59, 0x46, 0x6a, 0x2b, 0x6c, 0x6c, 0x62, 0x75, 0x49, 0x31, 0x5a, 0x79,
	0x67, 0x64, 0x78, 0x47, 0x73, 0x58, 0x38, 0x2b, 0x50, 0x33, 0x62, 0x79, 0x4d, 0x45, 0x41, 0x35,
	0x69, 0x67, 0x3d, 0x3d, 0x22, 0x7d, 0x22, 0x80, 0x04, 0x0a, 0x0b, 0x42, 0x69, 0x64, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x51, 0x0a, 0x0a, 0x62, 0x69, 0x64, 0x5f, 0x64, 0x69,
	0x67, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x42, 0x32, 0x92, 0x41, 0x2f, 0x32,
	0x2d, 0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x62,
	0x69, 0x64, 0x20, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x20, 0x73, 0x69, 0x67, 0x6e, 0x65,
	0x64, 0x20, 0x62, 0x79, 0x20, 0x74, 0x68, 0x65, 0x20, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x52, 0x09,
	0x62, 0x69, 0x64, 0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0x12, 0x57, 0x0a, 0x06, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x26, 0x2e, 0x72, 0x70, 0x63, 0x2e,
	0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x42,
	0x69, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x42, 0x17, 0x92, 0x41, 0x14, 0x32, 0x12, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x20, 0x6f,
	0x66, 0x20, 0x74, 0x68, 0x65, 0x20, 0x62, 0x69, 0x64, 0x2e, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x22, 0x4a, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x16, 0x0a, 0x12,
	0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49,
	0x45, 0x44, 0x10, 0x00, 0x12, 0x13, 0x0a, 0x0f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x41,
	0x43, 0x43, 0x45, 0x50, 0x54, 0x45, 0x44, 0x10, 0x01, 0x12, 0x13, 0x0a, 0x0f, 0x53, 0x54, 0x41,
	0x54, 0x55, 0x53, 0x5f, 0x52, 0x45, 0x4a, 0x45, 0x43, 0x54, 0x45, 0x44, 0x10, 0x02, 0x3a, 0xf8,
	0x01, 0x92, 0x41, 0xf4, 0x01, 0x0a, 0x69, 0x2a, 0x0c, 0x42, 0x69, 0x64, 0x20, 0x72, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x44, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x20,
	0x73, 0x65, 0x6e, 0x74, 0x20, 0x62, 0x79, 0x20, 0x74, 0x68, 0x65, 0x20, 0x70, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x20, 0x77, 0x69, 0x74, 0x68, 0x20, 0x74, 0x68, 0x65, 0x20, 0x64, 0x65,
	0x63, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x20, 0x6f, 0x6e, 0x20, 0x74, 0x68, 0x65, 0x20, 0x62, 0x69,
	0x64, 0x20, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x64, 0x2e, 0xd2, 0x01, 0x09, 0x62, 0x69,
	0x64, 0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0xd2, 0x01, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x32, 0x86, 0x01, 0x7b, 0x22, 0x62, 0x69, 0x64, 0x44, 0x69, 0x67, 0x65, 0x73, 0x74, 0x22, 0x3a,
	0x20, 0x22, 0x39, 0x64, 0x4a, 0x69, 0x6e, 0x77, 0x4c, 0x2b, 0x46, 0x5a, 0x36, 0x42, 0x31, 0x78,
	0x73, 0x49, 0x51, 0x51, 0x6f, 0x38, 0x74, 0x38, 0x42, 0x30, 0x5a, 0x58, 0x4a, 0x75, 0x62, 0x4a,
	0x77, 0x59, 0x38, 0x36, 0x6c, 0x2f, 0x59, 0x75, 0x37, 0x79, 0x41, 0x48, 0x31, 0x35, 0x39, 0x51,
	0x72, 0x50, 0x48, 0x55, 0x30, 0x71, 0x6a, 0x32, 0x50, 0x2b, 0x59, 0x46, 0x6a, 0x2b, 0x6c, 0x6c,
	0x62, 0x75, 0x49, 0x31, 0x5a, 0x79, 0x67, 0x64, 0x78, 0x47, 0x73, 0x58, 0x38, 0x2b, 0x50, 0x33,
	0x62, 0x79, 0x4d, 0x45, 0x41, 0x35, 0x69, 0x67, 0x3d, 0x3d, 0x22, 0x2c, 0x20, 0x22, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x22, 0x3a, 0x20, 0x22, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x41,
	0x43, 0x43, 0x45, 0x50, 0x54, 0x45, 0x44, 0x22, 0x7d, 0x32, 0xef, 0x04, 0x0a, 0x08, 0x50, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x12, 0x6d, 0x0a, 0x0b, 0x52, 0x65, 0x63, 0x65, 0x69, 0x76,
	0x65, 0x42, 0x69, 0x64, 0x73, 0x12, 0x20, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x17, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x69, 0x64,
	0x22, 0x21, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1b, 0x12, 0x19, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x5f, 0x62,
	0x69, 0x64, 0x73, 0x30, 0x01, 0x12, 0x85, 0x01, 0x0a, 0x11, 0x53, 0x65, 0x6e, 0x64, 0x50, 0x72,
	0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64, 0x42, 0x69, 0x64, 0x73, 0x12, 0x1f, 0x2e, 0x72, 0x70,
	0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x42, 0x69, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x1a, 0x20, 0x2e, 0x72,
	0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x2b,
	0x82, 0xd3, 0xe4, 0x93, 0x02, 0x25, 0x3a, 0x01, 0x2a, 0x22, 0x20, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x73, 0x65, 0x6e, 0x64, 0x5f, 0x70, 0x72, 0x6f,
	0x63, 0x65, 0x73, 0x73, 0x65, 0x64, 0x5f, 0x62, 0x69, 0x64, 0x73, 0x28, 0x01, 0x12, 0x82, 0x01,
	0x0a, 0x0d, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x12,
	0x20, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x21, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x2c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x26, 0x22, 0x24, 0x2f, 0x76,
	0x31, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6b, 0x65, 0x2f, 0x7b, 0x61, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0x7d, 0x12, 0x6f, 0x0a, 0x08, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x12, 0x20,
	0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61, 0x70, 0x69,
	0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x1a, 0x21, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12, 0x16, 0x2f, 0x76, 0x31,
	0x2f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x67, 0x65, 0x74, 0x5f, 0x73, 0x74,
	0x61, 0x6b, 0x65, 0x12, 0x76, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x4d, 0x69, 0x6e, 0x53, 0x74, 0x61,
	0x6b, 0x65, 0x12, 0x20, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x1a, 0x21, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69,
	0x64, 0x65, 0x72, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x22, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1c, 0x12,
	0x1a, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x2f, 0x67, 0x65,
	0x74, 0x5f, 0x6d, 0x69, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x6b, 0x65, 0x42, 0xc6, 0x01, 0x92, 0x41,
	0x7c, 0x12, 0x7a, 0x0a, 0x0c, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x20, 0x41, 0x50,
	0x49, 0x2a, 0x5d, 0x0a, 0x1b, 0x42, 0x75, 0x73, 0x69, 0x6e, 0x65, 0x73, 0x73, 0x20, 0x53, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x20, 0x4c, 0x69, 0x63, 0x65, 0x6e, 0x73, 0x65, 0x20, 0x31, 0x2e, 0x31,
	0x12, 0x3e, 0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x72, 0x69, 0x6d, 0x65, 0x76, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x2f, 0x6d, 0x65, 0x76, 0x2d, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x2f, 0x62,
	0x6c, 0x6f, 0x62, 0x2f, 0x6d, 0x61, 0x69, 0x6e, 0x2f, 0x4c, 0x49, 0x43, 0x45, 0x4e, 0x53, 0x45,
	0x32, 0x0b, 0x31, 0x2e, 0x30, 0x2e, 0x30, 0x2d, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x5a, 0x45, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x72, 0x69, 0x6d, 0x65, 0x76,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0x6d, 0x65, 0x76, 0x2d, 0x63, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x3b, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x61,
	0x70, 0x69, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_rpc_providerapi_v1_providerapi_proto_rawDescOnce sync.Once
	file_rpc_providerapi_v1_providerapi_proto_rawDescData = file_rpc_providerapi_v1_providerapi_proto_rawDesc
)

func file_rpc_providerapi_v1_providerapi_proto_rawDescGZIP() []byte {
	file_rpc_providerapi_v1_providerapi_proto_rawDescOnce.Do(func() {
		file_rpc_providerapi_v1_providerapi_proto_rawDescData = protoimpl.X.CompressGZIP(file_rpc_providerapi_v1_providerapi_proto_rawDescData)
	})
	return file_rpc_providerapi_v1_providerapi_proto_rawDescData
}

var file_rpc_providerapi_v1_providerapi_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_rpc_providerapi_v1_providerapi_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_rpc_providerapi_v1_providerapi_proto_goTypes = []interface{}{
	(BidResponse_Status)(0), // 0: rpc.providerapi.v1.BidResponse.Status
	(*StakeRequest)(nil),    // 1: rpc.providerapi.v1.StakeRequest
	(*StakeResponse)(nil),   // 2: rpc.providerapi.v1.StakeResponse
	(*EmptyMessage)(nil),    // 3: rpc.providerapi.v1.EmptyMessage
	(*Bid)(nil),             // 4: rpc.providerapi.v1.Bid
	(*BidResponse)(nil),     // 5: rpc.providerapi.v1.BidResponse
}
var file_rpc_providerapi_v1_providerapi_proto_depIdxs = []int32{
	0, // 0: rpc.providerapi.v1.BidResponse.status:type_name -> rpc.providerapi.v1.BidResponse.Status
	3, // 1: rpc.providerapi.v1.Provider.ReceiveBids:input_type -> rpc.providerapi.v1.EmptyMessage
	5, // 2: rpc.providerapi.v1.Provider.SendProcessedBids:input_type -> rpc.providerapi.v1.BidResponse
	1, // 3: rpc.providerapi.v1.Provider.RegisterStake:input_type -> rpc.providerapi.v1.StakeRequest
	3, // 4: rpc.providerapi.v1.Provider.GetStake:input_type -> rpc.providerapi.v1.EmptyMessage
	3, // 5: rpc.providerapi.v1.Provider.GetMinStake:input_type -> rpc.providerapi.v1.EmptyMessage
	4, // 6: rpc.providerapi.v1.Provider.ReceiveBids:output_type -> rpc.providerapi.v1.Bid
	3, // 7: rpc.providerapi.v1.Provider.SendProcessedBids:output_type -> rpc.providerapi.v1.EmptyMessage
	2, // 8: rpc.providerapi.v1.Provider.RegisterStake:output_type -> rpc.providerapi.v1.StakeResponse
	2, // 9: rpc.providerapi.v1.Provider.GetStake:output_type -> rpc.providerapi.v1.StakeResponse
	2, // 10: rpc.providerapi.v1.Provider.GetMinStake:output_type -> rpc.providerapi.v1.StakeResponse
	6, // [6:11] is the sub-list for method output_type
	1, // [1:6] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_rpc_providerapi_v1_providerapi_proto_init() }
func file_rpc_providerapi_v1_providerapi_proto_init() {
	if File_rpc_providerapi_v1_providerapi_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_rpc_providerapi_v1_providerapi_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StakeRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpc_providerapi_v1_providerapi_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StakeResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpc_providerapi_v1_providerapi_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EmptyMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpc_providerapi_v1_providerapi_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Bid); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_rpc_providerapi_v1_providerapi_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BidResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_rpc_providerapi_v1_providerapi_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_rpc_providerapi_v1_providerapi_proto_goTypes,
		DependencyIndexes: file_rpc_providerapi_v1_providerapi_proto_depIdxs,
		EnumInfos:         file_rpc_providerapi_v1_providerapi_proto_enumTypes,
		MessageInfos:      file_rpc_providerapi_v1_providerapi_proto_msgTypes,
	}.Build()
	File_rpc_providerapi_v1_providerapi_proto = out.File
	file_rpc_providerapi_v1_providerapi_proto_rawDesc = nil
	file_rpc_providerapi_v1_providerapi_proto_goTypes = nil
	file_rpc_providerapi_v1_providerapi_proto_depIdxs = nil
}
