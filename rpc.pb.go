// Code generated by protoc-gen-go. DO NOT EDIT.
// source: rpc.proto

/*
Package fidias is a generated protocol buffer package.

It is generated from these files:
	rpc.proto

It has these top-level messages:
	KVPair
	ReadStats
	WriteStats
	WriteOptions
	WriteRequest
	Request
	WriteResponse
*/
package fidias

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import hexatype "github.com/hexablock/hexatype"
import hexalog "github.com/hexablock/hexalog"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type KVPair struct {
	Key []byte `protobuf:"bytes,1,opt,name=Key,proto3" json:"Key,omitempty"`
	// Arbitrary data
	Value []byte `protobuf:"bytes,2,opt,name=Value,proto3" json:"Value,omitempty"`
	// Artibrary integer flags
	Flags int64 `protobuf:"varint,3,opt,name=Flags" json:"Flags,omitempty"`
	// Lamport time
	LTime uint64 `protobuf:"varint,4,opt,name=LTime" json:"LTime,omitempty"`
	// Modification time
	ModTime uint64 `protobuf:"varint,5,opt,name=ModTime" json:"ModTime,omitempty"`
	// Entry id resulting in the view.  If a directory is created then the dir
	// entry will also for  the same modification id as such directories will
	// have different values based on their view
	Modification []byte `protobuf:"bytes,6,opt,name=Modification,proto3" json:"Modification,omitempty"`
	// Entry height creating this view
	Height uint32 `protobuf:"varint,7,opt,name=Height" json:"Height,omitempty"`
}

func (m *KVPair) Reset()                    { *m = KVPair{} }
func (m *KVPair) String() string            { return proto.CompactTextString(m) }
func (*KVPair) ProtoMessage()               {}
func (*KVPair) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *KVPair) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *KVPair) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *KVPair) GetFlags() int64 {
	if m != nil {
		return m.Flags
	}
	return 0
}

func (m *KVPair) GetLTime() uint64 {
	if m != nil {
		return m.LTime
	}
	return 0
}

func (m *KVPair) GetModTime() uint64 {
	if m != nil {
		return m.ModTime
	}
	return 0
}

func (m *KVPair) GetModification() []byte {
	if m != nil {
		return m.Modification
	}
	return nil
}

func (m *KVPair) GetHeight() uint32 {
	if m != nil {
		return m.Height
	}
	return 0
}

type ReadStats struct {
	// Node serving the read
	Nodes []*hexatype.Node `protobuf:"bytes,1,rep,name=Nodes" json:"Nodes,omitempty"`
	// Affinity group
	Group int32 `protobuf:"varint,2,opt,name=Group" json:"Group,omitempty"`
	// Node priority in the group
	Priority int32 `protobuf:"varint,3,opt,name=Priority" json:"Priority,omitempty"`
	// Response time
	RespTime int64 `protobuf:"varint,4,opt,name=RespTime" json:"RespTime,omitempty"`
}

func (m *ReadStats) Reset()                    { *m = ReadStats{} }
func (m *ReadStats) String() string            { return proto.CompactTextString(m) }
func (*ReadStats) ProtoMessage()               {}
func (*ReadStats) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *ReadStats) GetNodes() []*hexatype.Node {
	if m != nil {
		return m.Nodes
	}
	return nil
}

func (m *ReadStats) GetGroup() int32 {
	if m != nil {
		return m.Group
	}
	return 0
}

func (m *ReadStats) GetPriority() int32 {
	if m != nil {
		return m.Priority
	}
	return 0
}

func (m *ReadStats) GetRespTime() int64 {
	if m != nil {
		return m.RespTime
	}
	return 0
}

type WriteStats struct {
	BallotTime   int64                  `protobuf:"varint,1,opt,name=BallotTime" json:"BallotTime,omitempty"`
	ApplyTime    int64                  `protobuf:"varint,2,opt,name=ApplyTime" json:"ApplyTime,omitempty"`
	Participants []*hexalog.Participant `protobuf:"bytes,3,rep,name=Participants" json:"Participants,omitempty"`
}

func (m *WriteStats) Reset()                    { *m = WriteStats{} }
func (m *WriteStats) String() string            { return proto.CompactTextString(m) }
func (*WriteStats) ProtoMessage()               {}
func (*WriteStats) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *WriteStats) GetBallotTime() int64 {
	if m != nil {
		return m.BallotTime
	}
	return 0
}

func (m *WriteStats) GetApplyTime() int64 {
	if m != nil {
		return m.ApplyTime
	}
	return 0
}

func (m *WriteStats) GetParticipants() []*hexalog.Participant {
	if m != nil {
		return m.Participants
	}
	return nil
}

type WriteOptions struct {
	WaitBallot       bool  `protobuf:"varint,1,opt,name=WaitBallot" json:"WaitBallot,omitempty"`
	WaitApply        bool  `protobuf:"varint,2,opt,name=WaitApply" json:"WaitApply,omitempty"`
	WaitApplyTimeout int64 `protobuf:"varint,3,opt,name=WaitApplyTimeout" json:"WaitApplyTimeout,omitempty"`
	Retries          int32 `protobuf:"varint,4,opt,name=Retries" json:"Retries,omitempty"`
	RetryInterval    int64 `protobuf:"varint,5,opt,name=RetryInterval" json:"RetryInterval,omitempty"`
}

func (m *WriteOptions) Reset()                    { *m = WriteOptions{} }
func (m *WriteOptions) String() string            { return proto.CompactTextString(m) }
func (*WriteOptions) ProtoMessage()               {}
func (*WriteOptions) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *WriteOptions) GetWaitBallot() bool {
	if m != nil {
		return m.WaitBallot
	}
	return false
}

func (m *WriteOptions) GetWaitApply() bool {
	if m != nil {
		return m.WaitApply
	}
	return false
}

func (m *WriteOptions) GetWaitApplyTimeout() int64 {
	if m != nil {
		return m.WaitApplyTimeout
	}
	return 0
}

func (m *WriteOptions) GetRetries() int32 {
	if m != nil {
		return m.Retries
	}
	return 0
}

func (m *WriteOptions) GetRetryInterval() int64 {
	if m != nil {
		return m.RetryInterval
	}
	return 0
}

type WriteRequest struct {
	KV      *KVPair       `protobuf:"bytes,1,opt,name=KV" json:"KV,omitempty"`
	Options *WriteOptions `protobuf:"bytes,2,opt,name=Options" json:"Options,omitempty"`
}

func (m *WriteRequest) Reset()                    { *m = WriteRequest{} }
func (m *WriteRequest) String() string            { return proto.CompactTextString(m) }
func (*WriteRequest) ProtoMessage()               {}
func (*WriteRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *WriteRequest) GetKV() *KVPair {
	if m != nil {
		return m.KV
	}
	return nil
}

func (m *WriteRequest) GetOptions() *WriteOptions {
	if m != nil {
		return m.Options
	}
	return nil
}

// Generic request
type Request struct {
}

func (m *Request) Reset()                    { *m = Request{} }
func (m *Request) String() string            { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()               {}
func (*Request) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

type WriteResponse struct {
	KV    *KVPair     `protobuf:"bytes,1,opt,name=KV" json:"KV,omitempty"`
	Stats *WriteStats `protobuf:"bytes,3,opt,name=Stats" json:"Stats,omitempty"`
}

func (m *WriteResponse) Reset()                    { *m = WriteResponse{} }
func (m *WriteResponse) String() string            { return proto.CompactTextString(m) }
func (*WriteResponse) ProtoMessage()               {}
func (*WriteResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *WriteResponse) GetKV() *KVPair {
	if m != nil {
		return m.KV
	}
	return nil
}

func (m *WriteResponse) GetStats() *WriteStats {
	if m != nil {
		return m.Stats
	}
	return nil
}

func init() {
	proto.RegisterType((*KVPair)(nil), "fidias.KVPair")
	proto.RegisterType((*ReadStats)(nil), "fidias.ReadStats")
	proto.RegisterType((*WriteStats)(nil), "fidias.WriteStats")
	proto.RegisterType((*WriteOptions)(nil), "fidias.WriteOptions")
	proto.RegisterType((*WriteRequest)(nil), "fidias.WriteRequest")
	proto.RegisterType((*Request)(nil), "fidias.Request")
	proto.RegisterType((*WriteResponse)(nil), "fidias.WriteResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for FidiasRPC service

type FidiasRPCClient interface {
	// Returns local node info
	LocalNodeRPC(ctx context.Context, in *Request, opts ...grpc.CallOption) (*hexatype.Node, error)
	// Get key-value pair from a single remote
	GetKeyRPC(ctx context.Context, in *KVPair, opts ...grpc.CallOption) (*KVPair, error)
	// List directory contents from a single remote
	ListDirRPC(ctx context.Context, in *KVPair, opts ...grpc.CallOption) (FidiasRPC_ListDirRPCClient, error)
	// Set key on cluster
	SetRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
	// Set key on cluster
	CASetRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
	// Remove key on cluster
	RemoveRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
	// Remove key on cluster
	CARemoveRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
}

type fidiasRPCClient struct {
	cc *grpc.ClientConn
}

func NewFidiasRPCClient(cc *grpc.ClientConn) FidiasRPCClient {
	return &fidiasRPCClient{cc}
}

func (c *fidiasRPCClient) LocalNodeRPC(ctx context.Context, in *Request, opts ...grpc.CallOption) (*hexatype.Node, error) {
	out := new(hexatype.Node)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/LocalNodeRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fidiasRPCClient) GetKeyRPC(ctx context.Context, in *KVPair, opts ...grpc.CallOption) (*KVPair, error) {
	out := new(KVPair)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/GetKeyRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fidiasRPCClient) ListDirRPC(ctx context.Context, in *KVPair, opts ...grpc.CallOption) (FidiasRPC_ListDirRPCClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_FidiasRPC_serviceDesc.Streams[0], c.cc, "/fidias.FidiasRPC/ListDirRPC", opts...)
	if err != nil {
		return nil, err
	}
	x := &fidiasRPCListDirRPCClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type FidiasRPC_ListDirRPCClient interface {
	Recv() (*KVPair, error)
	grpc.ClientStream
}

type fidiasRPCListDirRPCClient struct {
	grpc.ClientStream
}

func (x *fidiasRPCListDirRPCClient) Recv() (*KVPair, error) {
	m := new(KVPair)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *fidiasRPCClient) SetRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/SetRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fidiasRPCClient) CASetRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/CASetRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fidiasRPCClient) RemoveRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/RemoveRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fidiasRPCClient) CARemoveRPC(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/CARemoveRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for FidiasRPC service

type FidiasRPCServer interface {
	// Returns local node info
	LocalNodeRPC(context.Context, *Request) (*hexatype.Node, error)
	// Get key-value pair from a single remote
	GetKeyRPC(context.Context, *KVPair) (*KVPair, error)
	// List directory contents from a single remote
	ListDirRPC(*KVPair, FidiasRPC_ListDirRPCServer) error
	// Set key on cluster
	SetRPC(context.Context, *WriteRequest) (*WriteResponse, error)
	// Set key on cluster
	CASetRPC(context.Context, *WriteRequest) (*WriteResponse, error)
	// Remove key on cluster
	RemoveRPC(context.Context, *WriteRequest) (*WriteResponse, error)
	// Remove key on cluster
	CARemoveRPC(context.Context, *WriteRequest) (*WriteResponse, error)
}

func RegisterFidiasRPCServer(s *grpc.Server, srv FidiasRPCServer) {
	s.RegisterService(&_FidiasRPC_serviceDesc, srv)
}

func _FidiasRPC_LocalNodeRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FidiasRPCServer).LocalNodeRPC(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fidias.FidiasRPC/LocalNodeRPC",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FidiasRPCServer).LocalNodeRPC(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _FidiasRPC_GetKeyRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(KVPair)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FidiasRPCServer).GetKeyRPC(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fidias.FidiasRPC/GetKeyRPC",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FidiasRPCServer).GetKeyRPC(ctx, req.(*KVPair))
	}
	return interceptor(ctx, in, info, handler)
}

func _FidiasRPC_ListDirRPC_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(KVPair)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(FidiasRPCServer).ListDirRPC(m, &fidiasRPCListDirRPCServer{stream})
}

type FidiasRPC_ListDirRPCServer interface {
	Send(*KVPair) error
	grpc.ServerStream
}

type fidiasRPCListDirRPCServer struct {
	grpc.ServerStream
}

func (x *fidiasRPCListDirRPCServer) Send(m *KVPair) error {
	return x.ServerStream.SendMsg(m)
}

func _FidiasRPC_SetRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FidiasRPCServer).SetRPC(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fidias.FidiasRPC/SetRPC",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FidiasRPCServer).SetRPC(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FidiasRPC_CASetRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FidiasRPCServer).CASetRPC(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fidias.FidiasRPC/CASetRPC",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FidiasRPCServer).CASetRPC(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FidiasRPC_RemoveRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FidiasRPCServer).RemoveRPC(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fidias.FidiasRPC/RemoveRPC",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FidiasRPCServer).RemoveRPC(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FidiasRPC_CARemoveRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FidiasRPCServer).CARemoveRPC(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fidias.FidiasRPC/CARemoveRPC",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FidiasRPCServer).CARemoveRPC(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _FidiasRPC_serviceDesc = grpc.ServiceDesc{
	ServiceName: "fidias.FidiasRPC",
	HandlerType: (*FidiasRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "LocalNodeRPC",
			Handler:    _FidiasRPC_LocalNodeRPC_Handler,
		},
		{
			MethodName: "GetKeyRPC",
			Handler:    _FidiasRPC_GetKeyRPC_Handler,
		},
		{
			MethodName: "SetRPC",
			Handler:    _FidiasRPC_SetRPC_Handler,
		},
		{
			MethodName: "CASetRPC",
			Handler:    _FidiasRPC_CASetRPC_Handler,
		},
		{
			MethodName: "RemoveRPC",
			Handler:    _FidiasRPC_RemoveRPC_Handler,
		},
		{
			MethodName: "CARemoveRPC",
			Handler:    _FidiasRPC_CARemoveRPC_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ListDirRPC",
			Handler:       _FidiasRPC_ListDirRPC_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "rpc.proto",
}

func init() { proto.RegisterFile("rpc.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 613 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x54, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0xad, 0xe3, 0x3a, 0x4d, 0xa6, 0x29, 0x54, 0xab, 0x82, 0xac, 0x08, 0x55, 0x91, 0x55, 0xa1,
	0x08, 0x84, 0x5b, 0xca, 0x81, 0x0f, 0x71, 0x29, 0x41, 0x2d, 0x28, 0x2d, 0x44, 0x5b, 0x94, 0x8a,
	0x0b, 0xd2, 0xd6, 0xd9, 0xa6, 0x2b, 0xdc, 0xac, 0xd9, 0x5d, 0x57, 0xf8, 0xc4, 0x85, 0x2b, 0x7f,
	0x86, 0x13, 0x3f, 0x0f, 0xed, 0xac, 0x9d, 0x26, 0x41, 0x08, 0x94, 0xdb, 0xbe, 0x37, 0x6f, 0x3c,
	0x6f, 0x3e, 0x12, 0x68, 0xaa, 0x2c, 0x89, 0x33, 0x25, 0x8d, 0x24, 0xf5, 0x0b, 0x31, 0x12, 0x4c,
	0xb7, 0x1f, 0x8e, 0x85, 0xb9, 0xcc, 0xcf, 0xe3, 0x44, 0x5e, 0xed, 0x5e, 0xf2, 0xaf, 0xec, 0x3c,
	0x95, 0xc9, 0x67, 0x7c, 0x99, 0x22, 0xe3, 0xbb, 0xda, 0xa8, 0x3c, 0x31, 0xda, 0x25, 0xb5, 0xef,
	0xff, 0x55, 0x9c, 0xca, 0xf1, 0xee, 0xf4, 0xe3, 0xd1, 0x4f, 0x0f, 0xea, 0xfd, 0xe1, 0x80, 0x09,
	0x45, 0x36, 0xc1, 0xef, 0xf3, 0x22, 0xf4, 0x3a, 0x5e, 0xb7, 0x45, 0xed, 0x93, 0x6c, 0x41, 0x30,
	0x64, 0x69, 0xce, 0xc3, 0x1a, 0x72, 0x0e, 0x58, 0xf6, 0x30, 0x65, 0x63, 0x1d, 0xfa, 0x1d, 0xaf,
	0xeb, 0x53, 0x07, 0x2c, 0x7b, 0xfc, 0x41, 0x5c, 0xf1, 0x70, 0xb5, 0xe3, 0x75, 0x57, 0xa9, 0x03,
	0x24, 0x84, 0xb5, 0x13, 0x39, 0x42, 0x3e, 0x40, 0xbe, 0x82, 0x24, 0x82, 0xd6, 0x89, 0x1c, 0x89,
	0x0b, 0x91, 0x30, 0x23, 0xe4, 0x24, 0xac, 0x63, 0x89, 0x39, 0x8e, 0xdc, 0x85, 0xfa, 0x1b, 0x2e,
	0xc6, 0x97, 0x26, 0x5c, 0xeb, 0x78, 0xdd, 0x0d, 0x5a, 0xa2, 0xe8, 0x1b, 0x34, 0x29, 0x67, 0xa3,
	0x53, 0xc3, 0x8c, 0x26, 0x3b, 0x10, 0xbc, 0x93, 0x23, 0xae, 0x43, 0xaf, 0xe3, 0x77, 0xd7, 0xf7,
	0x6f, 0xc5, 0xd5, 0x44, 0x62, 0x4b, 0x53, 0x17, 0xb4, 0xf6, 0x8e, 0x94, 0xcc, 0x33, 0x6c, 0x25,
	0xa0, 0x0e, 0x90, 0x36, 0x34, 0x06, 0x4a, 0x48, 0x25, 0x4c, 0x81, 0xdd, 0x04, 0x74, 0x8a, 0x6d,
	0x8c, 0x72, 0x9d, 0x4d, 0x7b, 0xf2, 0xe9, 0x14, 0x47, 0xdf, 0x3d, 0x80, 0x33, 0x25, 0x0c, 0x77,
	0x16, 0xb6, 0x01, 0x5e, 0xb1, 0x34, 0x95, 0x06, 0xc5, 0x1e, 0x8a, 0x67, 0x18, 0x72, 0x0f, 0x9a,
	0x07, 0x59, 0x96, 0x16, 0x18, 0xae, 0x61, 0xf8, 0x86, 0x20, 0xcf, 0xa0, 0x35, 0x60, 0xca, 0x88,
	0x44, 0x64, 0x6c, 0x62, 0xec, 0x58, 0x6d, 0x1f, 0x5b, 0x71, 0xb9, 0xac, 0x78, 0x26, 0x48, 0xe7,
	0x94, 0xd1, 0x2f, 0x0f, 0x5a, 0x68, 0xe3, 0x7d, 0x66, 0xe7, 0x85, 0x46, 0xce, 0x98, 0x30, 0xae,
	0x34, 0x1a, 0x69, 0xd0, 0x19, 0xc6, 0x1a, 0xb1, 0x08, 0x6b, 0xa3, 0x91, 0x06, 0xbd, 0x21, 0xc8,
	0x03, 0xd8, 0x9c, 0x02, 0xeb, 0x4c, 0xe6, 0xa6, 0xdc, 0xf1, 0x1f, 0xbc, 0x5d, 0x2c, 0xe5, 0x46,
	0x09, 0xae, 0x71, 0x38, 0x01, 0xad, 0x20, 0xd9, 0x81, 0x0d, 0xfb, 0x2c, 0xde, 0x4e, 0x0c, 0x57,
	0xd7, 0x2c, 0xc5, 0xc5, 0xfb, 0x74, 0x9e, 0x8c, 0x3e, 0x95, 0xce, 0x29, 0xff, 0x92, 0x73, 0x6d,
	0xc8, 0x36, 0xd4, 0xfa, 0x43, 0x74, 0x6c, 0x57, 0xe8, 0x2e, 0x3e, 0x76, 0x87, 0x49, 0x6b, 0xfd,
	0x21, 0x89, 0x61, 0xad, 0x6c, 0x12, 0x7d, 0xdb, 0xf9, 0x94, 0xa2, 0xd9, 0x01, 0xd0, 0x4a, 0x14,
	0x35, 0xad, 0x3f, 0xfc, 0x74, 0xf4, 0x11, 0x36, 0xca, 0x52, 0x3a, 0x93, 0x13, 0xcd, 0xff, 0x59,
	0xab, 0x0b, 0x01, 0xee, 0x15, 0x9b, 0x5f, 0xdf, 0x27, 0x73, 0x95, 0x30, 0x42, 0x9d, 0x60, 0xff,
	0x87, 0x0f, 0xcd, 0x43, 0x0c, 0xd2, 0x41, 0x8f, 0x3c, 0x86, 0xd6, 0xb1, 0x4c, 0x58, 0x8a, 0x77,
	0x37, 0xe8, 0x91, 0xdb, 0x55, 0x62, 0xe9, 0xa4, 0xbd, 0x70, 0x9b, 0xd1, 0x0a, 0x79, 0x04, 0xcd,
	0x23, 0x6e, 0xfa, 0xbc, 0xb0, 0xfa, 0x05, 0x2f, 0xed, 0x05, 0x1c, 0xad, 0x90, 0x3d, 0x80, 0x63,
	0xa1, 0xcd, 0x6b, 0xa1, 0xfe, 0x4b, 0xbf, 0xe7, 0x91, 0xa7, 0x50, 0x3f, 0xe5, 0xc6, 0xaa, 0xe7,
	0x07, 0x56, 0x59, 0xba, 0xb3, 0xc0, 0xba, 0x11, 0x45, 0x2b, 0xe4, 0x39, 0x34, 0x7a, 0x07, 0xcb,
	0xa5, 0xbe, 0xb0, 0x3f, 0xcf, 0x2b, 0x79, 0xcd, 0x97, 0xc8, 0x7d, 0x09, 0xeb, 0xbd, 0x83, 0x65,
	0xb3, 0xcf, 0xeb, 0xf8, 0xa7, 0xf6, 0xe4, 0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0x23, 0x0d, 0x5c,
	0xd3, 0x3e, 0x05, 0x00, 0x00,
}
