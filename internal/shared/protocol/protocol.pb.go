// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        v5.29.2
// source: internal/shared/protocol/protocol.proto

package __

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NetworkMessage struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to MessageType:
	//
	//	*NetworkMessage_Announce
	//	*NetworkMessage_PeerListRequest
	//	*NetworkMessage_PeerListResponse
	//	*NetworkMessage_ChunkRequest
	//	*NetworkMessage_ChunkResponse
	//	*NetworkMessage_Heartbeat
	MessageType   isNetworkMessage_MessageType `protobuf_oneof:"message_type"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NetworkMessage) Reset() {
	*x = NetworkMessage{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NetworkMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NetworkMessage) ProtoMessage() {}

func (x *NetworkMessage) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NetworkMessage.ProtoReflect.Descriptor instead.
func (*NetworkMessage) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{0}
}

func (x *NetworkMessage) GetMessageType() isNetworkMessage_MessageType {
	if x != nil {
		return x.MessageType
	}
	return nil
}

func (x *NetworkMessage) GetAnnounce() *AnnounceMessage {
	if x != nil {
		if x, ok := x.MessageType.(*NetworkMessage_Announce); ok {
			return x.Announce
		}
	}
	return nil
}

func (x *NetworkMessage) GetPeerListRequest() *PeerListRequest {
	if x != nil {
		if x, ok := x.MessageType.(*NetworkMessage_PeerListRequest); ok {
			return x.PeerListRequest
		}
	}
	return nil
}

func (x *NetworkMessage) GetPeerListResponse() *PeerListResponse {
	if x != nil {
		if x, ok := x.MessageType.(*NetworkMessage_PeerListResponse); ok {
			return x.PeerListResponse
		}
	}
	return nil
}

func (x *NetworkMessage) GetChunkRequest() *ChunkRequest {
	if x != nil {
		if x, ok := x.MessageType.(*NetworkMessage_ChunkRequest); ok {
			return x.ChunkRequest
		}
	}
	return nil
}

func (x *NetworkMessage) GetChunkResponse() *ChunkResponse {
	if x != nil {
		if x, ok := x.MessageType.(*NetworkMessage_ChunkResponse); ok {
			return x.ChunkResponse
		}
	}
	return nil
}

func (x *NetworkMessage) GetHeartbeat() *HeartbeatMessage {
	if x != nil {
		if x, ok := x.MessageType.(*NetworkMessage_Heartbeat); ok {
			return x.Heartbeat
		}
	}
	return nil
}

type isNetworkMessage_MessageType interface {
	isNetworkMessage_MessageType()
}

type NetworkMessage_Announce struct {
	Announce *AnnounceMessage `protobuf:"bytes,1,opt,name=announce,proto3,oneof"`
}

type NetworkMessage_PeerListRequest struct {
	PeerListRequest *PeerListRequest `protobuf:"bytes,2,opt,name=peer_list_request,json=peerListRequest,proto3,oneof"`
}

type NetworkMessage_PeerListResponse struct {
	PeerListResponse *PeerListResponse `protobuf:"bytes,3,opt,name=peer_list_response,json=peerListResponse,proto3,oneof"`
}

type NetworkMessage_ChunkRequest struct {
	ChunkRequest *ChunkRequest `protobuf:"bytes,4,opt,name=chunk_request,json=chunkRequest,proto3,oneof"`
}

type NetworkMessage_ChunkResponse struct {
	ChunkResponse *ChunkResponse `protobuf:"bytes,5,opt,name=chunk_response,json=chunkResponse,proto3,oneof"`
}

type NetworkMessage_Heartbeat struct {
	Heartbeat *HeartbeatMessage `protobuf:"bytes,6,opt,name=heartbeat,proto3,oneof"`
}

func (*NetworkMessage_Announce) isNetworkMessage_MessageType() {}

func (*NetworkMessage_PeerListRequest) isNetworkMessage_MessageType() {}

func (*NetworkMessage_PeerListResponse) isNetworkMessage_MessageType() {}

func (*NetworkMessage_ChunkRequest) isNetworkMessage_MessageType() {}

func (*NetworkMessage_ChunkResponse) isNetworkMessage_MessageType() {}

func (*NetworkMessage_Heartbeat) isNetworkMessage_MessageType() {}

type PeerInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	PeerId        string                 `protobuf:"bytes,1,opt,name=peer_id,json=peerId,proto3" json:"peer_id,omitempty"`
	IpAddress     string                 `protobuf:"bytes,2,opt,name=ip_address,json=ipAddress,proto3" json:"ip_address,omitempty"`
	Port          int32                  `protobuf:"varint,3,opt,name=port,proto3" json:"port,omitempty"`
	LastHeartbeat int64                  `protobuf:"varint,4,opt,name=last_heartbeat,json=lastHeartbeat,proto3" json:"last_heartbeat,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PeerInfo) Reset() {
	*x = PeerInfo{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PeerInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PeerInfo) ProtoMessage() {}

func (x *PeerInfo) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PeerInfo.ProtoReflect.Descriptor instead.
func (*PeerInfo) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{1}
}

func (x *PeerInfo) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

func (x *PeerInfo) GetIpAddress() string {
	if x != nil {
		return x.IpAddress
	}
	return ""
}

func (x *PeerInfo) GetPort() int32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *PeerInfo) GetLastHeartbeat() int64 {
	if x != nil {
		return x.LastHeartbeat
	}
	return 0
}

type FileInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileHash      string                 `protobuf:"bytes,1,opt,name=file_hash,json=fileHash,proto3" json:"file_hash,omitempty"`
	TotalChunks   int32                  `protobuf:"varint,2,opt,name=total_chunks,json=totalChunks,proto3" json:"total_chunks,omitempty"`
	ChunkSize     int32                  `protobuf:"varint,3,opt,name=chunk_size,json=chunkSize,proto3" json:"chunk_size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FileInfo) Reset() {
	*x = FileInfo{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileInfo) ProtoMessage() {}

func (x *FileInfo) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileInfo.ProtoReflect.Descriptor instead.
func (*FileInfo) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{2}
}

func (x *FileInfo) GetFileHash() string {
	if x != nil {
		return x.FileHash
	}
	return ""
}

func (x *FileInfo) GetTotalChunks() int32 {
	if x != nil {
		return x.TotalChunks
	}
	return 0
}

func (x *FileInfo) GetChunkSize() int32 {
	if x != nil {
		return x.ChunkSize
	}
	return 0
}

type AnnounceMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Peer          *PeerInfo              `protobuf:"bytes,2,opt,name=peer,proto3" json:"peer,omitempty"`
	Files         []*FileInfo            `protobuf:"bytes,3,rep,name=files,proto3" json:"files,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AnnounceMessage) Reset() {
	*x = AnnounceMessage{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AnnounceMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AnnounceMessage) ProtoMessage() {}

func (x *AnnounceMessage) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AnnounceMessage.ProtoReflect.Descriptor instead.
func (*AnnounceMessage) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{3}
}

func (x *AnnounceMessage) GetPeer() *PeerInfo {
	if x != nil {
		return x.Peer
	}
	return nil
}

func (x *AnnounceMessage) GetFiles() []*FileInfo {
	if x != nil {
		return x.Files
	}
	return nil
}

type PeerListRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileHash      string                 `protobuf:"bytes,2,opt,name=file_hash,json=fileHash,proto3" json:"file_hash,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PeerListRequest) Reset() {
	*x = PeerListRequest{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PeerListRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PeerListRequest) ProtoMessage() {}

func (x *PeerListRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PeerListRequest.ProtoReflect.Descriptor instead.
func (*PeerListRequest) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{4}
}

func (x *PeerListRequest) GetFileHash() string {
	if x != nil {
		return x.FileHash
	}
	return ""
}

type PeerListResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileHash      string                 `protobuf:"bytes,2,opt,name=file_hash,json=fileHash,proto3" json:"file_hash,omitempty"`
	TotalChunks   int32                  `protobuf:"varint,3,opt,name=total_chunks,json=totalChunks,proto3" json:"total_chunks,omitempty"`
	ChunkSize     int32                  `protobuf:"varint,4,opt,name=chunk_size,json=chunkSize,proto3" json:"chunk_size,omitempty"`
	Peers         []*PeerInfo            `protobuf:"bytes,5,rep,name=peers,proto3" json:"peers,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PeerListResponse) Reset() {
	*x = PeerListResponse{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PeerListResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PeerListResponse) ProtoMessage() {}

func (x *PeerListResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PeerListResponse.ProtoReflect.Descriptor instead.
func (*PeerListResponse) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{5}
}

func (x *PeerListResponse) GetFileHash() string {
	if x != nil {
		return x.FileHash
	}
	return ""
}

func (x *PeerListResponse) GetTotalChunks() int32 {
	if x != nil {
		return x.TotalChunks
	}
	return 0
}

func (x *PeerListResponse) GetChunkSize() int32 {
	if x != nil {
		return x.ChunkSize
	}
	return 0
}

func (x *PeerListResponse) GetPeers() []*PeerInfo {
	if x != nil {
		return x.Peers
	}
	return nil
}

type ChunkRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileHash      string                 `protobuf:"bytes,2,opt,name=file_hash,json=fileHash,proto3" json:"file_hash,omitempty"`
	ChunkIndex    int32                  `protobuf:"varint,3,opt,name=chunk_index,json=chunkIndex,proto3" json:"chunk_index,omitempty"`
	ChunkHash     string                 `protobuf:"bytes,4,opt,name=chunk_hash,json=chunkHash,proto3" json:"chunk_hash,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChunkRequest) Reset() {
	*x = ChunkRequest{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChunkRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChunkRequest) ProtoMessage() {}

func (x *ChunkRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChunkRequest.ProtoReflect.Descriptor instead.
func (*ChunkRequest) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{6}
}

func (x *ChunkRequest) GetFileHash() string {
	if x != nil {
		return x.FileHash
	}
	return ""
}

func (x *ChunkRequest) GetChunkIndex() int32 {
	if x != nil {
		return x.ChunkIndex
	}
	return 0
}

func (x *ChunkRequest) GetChunkHash() string {
	if x != nil {
		return x.ChunkHash
	}
	return ""
}

type ChunkResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileHash      string                 `protobuf:"bytes,2,opt,name=file_hash,json=fileHash,proto3" json:"file_hash,omitempty"`
	ChunkIndex    int32                  `protobuf:"varint,3,opt,name=chunk_index,json=chunkIndex,proto3" json:"chunk_index,omitempty"`
	ChunkData     []byte                 `protobuf:"bytes,4,opt,name=chunk_data,json=chunkData,proto3" json:"chunk_data,omitempty"`
	ChunkHash     string                 `protobuf:"bytes,5,opt,name=chunk_hash,json=chunkHash,proto3" json:"chunk_hash,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChunkResponse) Reset() {
	*x = ChunkResponse{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChunkResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChunkResponse) ProtoMessage() {}

func (x *ChunkResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChunkResponse.ProtoReflect.Descriptor instead.
func (*ChunkResponse) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{7}
}

func (x *ChunkResponse) GetFileHash() string {
	if x != nil {
		return x.FileHash
	}
	return ""
}

func (x *ChunkResponse) GetChunkIndex() int32 {
	if x != nil {
		return x.ChunkIndex
	}
	return 0
}

func (x *ChunkResponse) GetChunkData() []byte {
	if x != nil {
		return x.ChunkData
	}
	return nil
}

func (x *ChunkResponse) GetChunkHash() string {
	if x != nil {
		return x.ChunkHash
	}
	return ""
}

type HeartbeatMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Timestamp     int64                  `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HeartbeatMessage) Reset() {
	*x = HeartbeatMessage{}
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HeartbeatMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeartbeatMessage) ProtoMessage() {}

func (x *HeartbeatMessage) ProtoReflect() protoreflect.Message {
	mi := &file_internal_shared_protocol_protocol_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeartbeatMessage.ProtoReflect.Descriptor instead.
func (*HeartbeatMessage) Descriptor() ([]byte, []int) {
	return file_internal_shared_protocol_protocol_proto_rawDescGZIP(), []int{8}
}

func (x *HeartbeatMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

var File_internal_shared_protocol_protocol_proto protoreflect.FileDescriptor

var file_internal_shared_protocol_protocol_proto_rawDesc = string([]byte{
	0x0a, 0x27, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x68, 0x61, 0x72, 0x65,
	0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x22, 0xab, 0x03, 0x0a, 0x0e, 0x4e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x37, 0x0a, 0x08, 0x61, 0x6e, 0x6e, 0x6f, 0x75, 0x6e,
	0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x2e, 0x41, 0x6e, 0x6e, 0x6f, 0x75, 0x6e, 0x63, 0x65, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x48, 0x00, 0x52, 0x08, 0x61, 0x6e, 0x6e, 0x6f, 0x75, 0x6e, 0x63, 0x65, 0x12,
	0x47, 0x0a, 0x11, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x50, 0x65, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x0f, 0x70, 0x65, 0x65, 0x72, 0x4c, 0x69, 0x73,
	0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x4a, 0x0a, 0x12, 0x70, 0x65, 0x65, 0x72,
	0x5f, 0x6c, 0x69, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e,
	0x50, 0x65, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x48, 0x00, 0x52, 0x10, 0x70, 0x65, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3d, 0x0a, 0x0d, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x0c, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x40, 0x0a, 0x0e, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x72, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x48, 0x00, 0x52, 0x0d, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3a, 0x0a, 0x09, 0x68, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65,
	0x61, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x2e, 0x48, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61, 0x74, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x48, 0x00, 0x52, 0x09, 0x68, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61,
	0x74, 0x42, 0x0e, 0x0a, 0x0c, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x79, 0x70,
	0x65, 0x22, 0x7d, 0x0a, 0x08, 0x50, 0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x17, 0x0a,
	0x07, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x69, 0x70, 0x5f, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x69, 0x70, 0x41, 0x64,
	0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x25, 0x0a, 0x0e, 0x6c, 0x61, 0x73,
	0x74, 0x5f, 0x68, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x0d, 0x6c, 0x61, 0x73, 0x74, 0x48, 0x65, 0x61, 0x72, 0x74, 0x62, 0x65, 0x61, 0x74,
	0x22, 0x69, 0x0a, 0x08, 0x46, 0x69, 0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1b, 0x0a, 0x09,
	0x66, 0x69, 0x6c, 0x65, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x66, 0x69, 0x6c, 0x65, 0x48, 0x61, 0x73, 0x68, 0x12, 0x21, 0x0a, 0x0c, 0x74, 0x6f, 0x74,
	0x61, 0x6c, 0x5f, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x0b, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x12, 0x1d, 0x0a, 0x0a,
	0x63, 0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x09, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x53, 0x69, 0x7a, 0x65, 0x22, 0x63, 0x0a, 0x0f, 0x41,
	0x6e, 0x6e, 0x6f, 0x75, 0x6e, 0x63, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x26,
	0x0a, 0x04, 0x70, 0x65, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x50, 0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f,
	0x52, 0x04, 0x70, 0x65, 0x65, 0x72, 0x12, 0x28, 0x0a, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x18,
	0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c,
	0x2e, 0x46, 0x69, 0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73,
	0x22, 0x2e, 0x0a, 0x0f, 0x50, 0x65, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x48, 0x61, 0x73, 0x68,
	0x22, 0x9b, 0x01, 0x0a, 0x10, 0x50, 0x65, 0x65, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x68, 0x61,
	0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x48, 0x61,
	0x73, 0x68, 0x12, 0x21, 0x0a, 0x0c, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x63, 0x68, 0x75, 0x6e,
	0x6b, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0b, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x43,
	0x68, 0x75, 0x6e, 0x6b, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x73,
	0x69, 0x7a, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x63, 0x68, 0x75, 0x6e, 0x6b,
	0x53, 0x69, 0x7a, 0x65, 0x12, 0x28, 0x0a, 0x05, 0x70, 0x65, 0x65, 0x72, 0x73, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2e, 0x50,
	0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x05, 0x70, 0x65, 0x65, 0x72, 0x73, 0x22, 0x6b,
	0x0a, 0x0c, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1b,
	0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x63,
	0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x0a, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x1d, 0x0a, 0x0a,
	0x63, 0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x22, 0x8b, 0x01, 0x0a, 0x0d,
	0x43, 0x68, 0x75, 0x6e, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1b, 0x0a,
	0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x68,
	0x75, 0x6e, 0x6b, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x0a, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x1d, 0x0a, 0x0a, 0x63,
	0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x44, 0x61, 0x74, 0x61, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x68,
	0x75, 0x6e, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x63, 0x68, 0x75, 0x6e, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x22, 0x30, 0x0a, 0x10, 0x48, 0x65, 0x61,
	0x72, 0x74, 0x62, 0x65, 0x61, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x1c, 0x0a,
	0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0x04, 0x5a, 0x02, 0x2e,
	0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_internal_shared_protocol_protocol_proto_rawDescOnce sync.Once
	file_internal_shared_protocol_protocol_proto_rawDescData []byte
)

func file_internal_shared_protocol_protocol_proto_rawDescGZIP() []byte {
	file_internal_shared_protocol_protocol_proto_rawDescOnce.Do(func() {
		file_internal_shared_protocol_protocol_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_internal_shared_protocol_protocol_proto_rawDesc), len(file_internal_shared_protocol_protocol_proto_rawDesc)))
	})
	return file_internal_shared_protocol_protocol_proto_rawDescData
}

var file_internal_shared_protocol_protocol_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_internal_shared_protocol_protocol_proto_goTypes = []any{
	(*NetworkMessage)(nil),   // 0: protocol.NetworkMessage
	(*PeerInfo)(nil),         // 1: protocol.PeerInfo
	(*FileInfo)(nil),         // 2: protocol.FileInfo
	(*AnnounceMessage)(nil),  // 3: protocol.AnnounceMessage
	(*PeerListRequest)(nil),  // 4: protocol.PeerListRequest
	(*PeerListResponse)(nil), // 5: protocol.PeerListResponse
	(*ChunkRequest)(nil),     // 6: protocol.ChunkRequest
	(*ChunkResponse)(nil),    // 7: protocol.ChunkResponse
	(*HeartbeatMessage)(nil), // 8: protocol.HeartbeatMessage
}
var file_internal_shared_protocol_protocol_proto_depIdxs = []int32{
	3, // 0: protocol.NetworkMessage.announce:type_name -> protocol.AnnounceMessage
	4, // 1: protocol.NetworkMessage.peer_list_request:type_name -> protocol.PeerListRequest
	5, // 2: protocol.NetworkMessage.peer_list_response:type_name -> protocol.PeerListResponse
	6, // 3: protocol.NetworkMessage.chunk_request:type_name -> protocol.ChunkRequest
	7, // 4: protocol.NetworkMessage.chunk_response:type_name -> protocol.ChunkResponse
	8, // 5: protocol.NetworkMessage.heartbeat:type_name -> protocol.HeartbeatMessage
	1, // 6: protocol.AnnounceMessage.peer:type_name -> protocol.PeerInfo
	2, // 7: protocol.AnnounceMessage.files:type_name -> protocol.FileInfo
	1, // 8: protocol.PeerListResponse.peers:type_name -> protocol.PeerInfo
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_internal_shared_protocol_protocol_proto_init() }
func file_internal_shared_protocol_protocol_proto_init() {
	if File_internal_shared_protocol_protocol_proto != nil {
		return
	}
	file_internal_shared_protocol_protocol_proto_msgTypes[0].OneofWrappers = []any{
		(*NetworkMessage_Announce)(nil),
		(*NetworkMessage_PeerListRequest)(nil),
		(*NetworkMessage_PeerListResponse)(nil),
		(*NetworkMessage_ChunkRequest)(nil),
		(*NetworkMessage_ChunkResponse)(nil),
		(*NetworkMessage_Heartbeat)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_internal_shared_protocol_protocol_proto_rawDesc), len(file_internal_shared_protocol_protocol_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_shared_protocol_protocol_proto_goTypes,
		DependencyIndexes: file_internal_shared_protocol_protocol_proto_depIdxs,
		MessageInfos:      file_internal_shared_protocol_protocol_proto_msgTypes,
	}.Build()
	File_internal_shared_protocol_protocol_proto = out.File
	file_internal_shared_protocol_protocol_proto_goTypes = nil
	file_internal_shared_protocol_protocol_proto_depIdxs = nil
}
