// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: preconfirmation/v1/preconfirmation.proto

package preconfirmationv1

import (
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

type Bid struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TxHash              string `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	BidAmount           string `protobuf:"bytes,2,opt,name=bid_amount,json=bidAmount,proto3" json:"bid_amount,omitempty"`
	BlockNumber         int64  `protobuf:"varint,3,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Digest              []byte `protobuf:"bytes,4,opt,name=digest,proto3" json:"digest,omitempty"`
	Signature           []byte `protobuf:"bytes,5,opt,name=signature,proto3" json:"signature,omitempty"`
	DecayStartTimestamp int64  `protobuf:"varint,6,opt,name=decay_start_timestamp,json=decayStartTimestamp,proto3" json:"decay_start_timestamp,omitempty"`
	DecayEndTimestamp   int64  `protobuf:"varint,7,opt,name=decay_end_timestamp,json=decayEndTimestamp,proto3" json:"decay_end_timestamp,omitempty"`
	NikePublicKey       []byte `protobuf:"bytes,8,opt,name=nike_public_key,json=nikePublicKey,proto3" json:"nike_public_key,omitempty"`
}

func (x *Bid) Reset() {
	*x = Bid{}
	if protoimpl.UnsafeEnabled {
		mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Bid) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bid) ProtoMessage() {}

func (x *Bid) ProtoReflect() protoreflect.Message {
	mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[0]
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
	return file_preconfirmation_v1_preconfirmation_proto_rawDescGZIP(), []int{0}
}

func (x *Bid) GetTxHash() string {
	if x != nil {
		return x.TxHash
	}
	return ""
}

func (x *Bid) GetBidAmount() string {
	if x != nil {
		return x.BidAmount
	}
	return ""
}

func (x *Bid) GetBlockNumber() int64 {
	if x != nil {
		return x.BlockNumber
	}
	return 0
}

func (x *Bid) GetDigest() []byte {
	if x != nil {
		return x.Digest
	}
	return nil
}

func (x *Bid) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *Bid) GetDecayStartTimestamp() int64 {
	if x != nil {
		return x.DecayStartTimestamp
	}
	return 0
}

func (x *Bid) GetDecayEndTimestamp() int64 {
	if x != nil {
		return x.DecayEndTimestamp
	}
	return 0
}

func (x *Bid) GetNikePublicKey() []byte {
	if x != nil {
		return x.NikePublicKey
	}
	return nil
}

type EncryptedBid struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ciphertext []byte `protobuf:"bytes,1,opt,name=ciphertext,proto3" json:"ciphertext,omitempty"`
}

func (x *EncryptedBid) Reset() {
	*x = EncryptedBid{}
	if protoimpl.UnsafeEnabled {
		mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EncryptedBid) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EncryptedBid) ProtoMessage() {}

func (x *EncryptedBid) ProtoReflect() protoreflect.Message {
	mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EncryptedBid.ProtoReflect.Descriptor instead.
func (*EncryptedBid) Descriptor() ([]byte, []int) {
	return file_preconfirmation_v1_preconfirmation_proto_rawDescGZIP(), []int{1}
}

func (x *EncryptedBid) GetCiphertext() []byte {
	if x != nil {
		return x.Ciphertext
	}
	return nil
}

type PreConfirmation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Bid             *Bid   `protobuf:"bytes,1,opt,name=bid,proto3" json:"bid,omitempty"`
	Digest          []byte `protobuf:"bytes,2,opt,name=digest,proto3" json:"digest,omitempty"`
	Signature       []byte `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
	ProviderAddress []byte `protobuf:"bytes,4,opt,name=provider_address,json=providerAddress,proto3" json:"provider_address,omitempty"`
	SharedSecret    []byte `protobuf:"bytes,5,opt,name=shared_secret,json=sharedSecret,proto3" json:"shared_secret,omitempty"`
}

func (x *PreConfirmation) Reset() {
	*x = PreConfirmation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PreConfirmation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PreConfirmation) ProtoMessage() {}

func (x *PreConfirmation) ProtoReflect() protoreflect.Message {
	mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PreConfirmation.ProtoReflect.Descriptor instead.
func (*PreConfirmation) Descriptor() ([]byte, []int) {
	return file_preconfirmation_v1_preconfirmation_proto_rawDescGZIP(), []int{2}
}

func (x *PreConfirmation) GetBid() *Bid {
	if x != nil {
		return x.Bid
	}
	return nil
}

func (x *PreConfirmation) GetDigest() []byte {
	if x != nil {
		return x.Digest
	}
	return nil
}

func (x *PreConfirmation) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *PreConfirmation) GetProviderAddress() []byte {
	if x != nil {
		return x.ProviderAddress
	}
	return nil
}

func (x *PreConfirmation) GetSharedSecret() []byte {
	if x != nil {
		return x.SharedSecret
	}
	return nil
}

type EncryptedPreConfirmation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Commitment []byte `protobuf:"bytes,1,opt,name=commitment,proto3" json:"commitment,omitempty"`
	Signature  []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *EncryptedPreConfirmation) Reset() {
	*x = EncryptedPreConfirmation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EncryptedPreConfirmation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EncryptedPreConfirmation) ProtoMessage() {}

func (x *EncryptedPreConfirmation) ProtoReflect() protoreflect.Message {
	mi := &file_preconfirmation_v1_preconfirmation_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EncryptedPreConfirmation.ProtoReflect.Descriptor instead.
func (*EncryptedPreConfirmation) Descriptor() ([]byte, []int) {
	return file_preconfirmation_v1_preconfirmation_proto_rawDescGZIP(), []int{3}
}

func (x *EncryptedPreConfirmation) GetCommitment() []byte {
	if x != nil {
		return x.Commitment
	}
	return nil
}

func (x *EncryptedPreConfirmation) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

var File_preconfirmation_v1_preconfirmation_proto protoreflect.FileDescriptor

var file_preconfirmation_v1_preconfirmation_proto_rawDesc = []byte{
	0x0a, 0x28, 0x70, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x70, 0x72, 0x65, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x22, 0xa2,
	0x02, 0x0a, 0x03, 0x42, 0x69, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x74, 0x78, 0x5f, 0x68, 0x61, 0x73,
	0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x74, 0x78, 0x48, 0x61, 0x73, 0x68, 0x12,
	0x1d, 0x0a, 0x0a, 0x62, 0x69, 0x64, 0x5f, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x62, 0x69, 0x64, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x21,
	0x0a, 0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e, 0x75, 0x6d, 0x62, 0x65,
	0x72, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67,
	0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69,
	0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x32, 0x0a, 0x15, 0x64, 0x65, 0x63, 0x61, 0x79,
	0x5f, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x13, 0x64, 0x65, 0x63, 0x61, 0x79, 0x53, 0x74, 0x61,
	0x72, 0x74, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x2e, 0x0a, 0x13, 0x64,
	0x65, 0x63, 0x61, 0x79, 0x5f, 0x65, 0x6e, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x11, 0x64, 0x65, 0x63, 0x61, 0x79, 0x45,
	0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x26, 0x0a, 0x0f, 0x6e,
	0x69, 0x6b, 0x65, 0x5f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x08,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x6e, 0x69, 0x6b, 0x65, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x4b, 0x65, 0x79, 0x22, 0x2e, 0x0a, 0x0c, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65, 0x64,
	0x42, 0x69, 0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x69, 0x70, 0x68, 0x65, 0x72, 0x74, 0x65, 0x78,
	0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x63, 0x69, 0x70, 0x68, 0x65, 0x72, 0x74,
	0x65, 0x78, 0x74, 0x22, 0xc2, 0x01, 0x0a, 0x0f, 0x50, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x29, 0x0a, 0x03, 0x62, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x70, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72,
	0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x69, 0x64, 0x52, 0x03, 0x62,
	0x69, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69,
	0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73,
	0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x29, 0x0a, 0x10, 0x70, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x0f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x41, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x73, 0x65,
	0x63, 0x72, 0x65, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x73, 0x68, 0x61, 0x72,
	0x65, 0x64, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x22, 0x58, 0x0a, 0x18, 0x45, 0x6e, 0x63, 0x72,
	0x79, 0x70, 0x74, 0x65, 0x64, 0x50, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x6d, 0x65,
	0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x6d, 0x65, 0x6e, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x42, 0xe9, 0x01, 0x0a, 0x16, 0x63, 0x6f, 0x6d, 0x2e, 0x70, 0x72, 0x65, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x76, 0x31, 0x42, 0x14, 0x50,
	0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x50, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x70, 0x72, 0x69, 0x6d, 0x65, 0x76, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c,
	0x2f, 0x6d, 0x65, 0x76, 0x2d, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x2f, 0x67, 0x65, 0x6e, 0x2f,
	0x67, 0x6f, 0x2f, 0x70, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2f, 0x76, 0x31, 0x3b, 0x70, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x50, 0x58, 0x58, 0xaa, 0x02, 0x12,
	0x50, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x56, 0x31, 0xca, 0x02, 0x12, 0x50, 0x72, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x1e, 0x50, 0x72, 0x65, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x13, 0x50, 0x72, 0x65, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_preconfirmation_v1_preconfirmation_proto_rawDescOnce sync.Once
	file_preconfirmation_v1_preconfirmation_proto_rawDescData = file_preconfirmation_v1_preconfirmation_proto_rawDesc
)

func file_preconfirmation_v1_preconfirmation_proto_rawDescGZIP() []byte {
	file_preconfirmation_v1_preconfirmation_proto_rawDescOnce.Do(func() {
		file_preconfirmation_v1_preconfirmation_proto_rawDescData = protoimpl.X.CompressGZIP(file_preconfirmation_v1_preconfirmation_proto_rawDescData)
	})
	return file_preconfirmation_v1_preconfirmation_proto_rawDescData
}

var file_preconfirmation_v1_preconfirmation_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_preconfirmation_v1_preconfirmation_proto_goTypes = []interface{}{
	(*Bid)(nil),                      // 0: preconfirmation.v1.Bid
	(*EncryptedBid)(nil),             // 1: preconfirmation.v1.EncryptedBid
	(*PreConfirmation)(nil),          // 2: preconfirmation.v1.PreConfirmation
	(*EncryptedPreConfirmation)(nil), // 3: preconfirmation.v1.EncryptedPreConfirmation
}
var file_preconfirmation_v1_preconfirmation_proto_depIdxs = []int32{
	0, // 0: preconfirmation.v1.PreConfirmation.bid:type_name -> preconfirmation.v1.Bid
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_preconfirmation_v1_preconfirmation_proto_init() }
func file_preconfirmation_v1_preconfirmation_proto_init() {
	if File_preconfirmation_v1_preconfirmation_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_preconfirmation_v1_preconfirmation_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_preconfirmation_v1_preconfirmation_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EncryptedBid); i {
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
		file_preconfirmation_v1_preconfirmation_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PreConfirmation); i {
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
		file_preconfirmation_v1_preconfirmation_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EncryptedPreConfirmation); i {
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
			RawDescriptor: file_preconfirmation_v1_preconfirmation_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_preconfirmation_v1_preconfirmation_proto_goTypes,
		DependencyIndexes: file_preconfirmation_v1_preconfirmation_proto_depIdxs,
		MessageInfos:      file_preconfirmation_v1_preconfirmation_proto_msgTypes,
	}.Build()
	File_preconfirmation_v1_preconfirmation_proto = out.File
	file_preconfirmation_v1_preconfirmation_proto_rawDesc = nil
	file_preconfirmation_v1_preconfirmation_proto_goTypes = nil
	file_preconfirmation_v1_preconfirmation_proto_depIdxs = nil
}
