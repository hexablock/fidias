// Code generated by protoc-gen-go.
// source: rpc.proto
// DO NOT EDIT!

/*
Package fidias is a generated protocol buffer package.

It is generated from these files:
	rpc.proto

It has these top-level messages:
	KeyValuePair
*/
package fidias

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
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

type KeyValuePair struct {
	Key   []byte         `protobuf:"bytes,1,opt,name=Key,json=key,proto3" json:"Key,omitempty"`
	Value []byte         `protobuf:"bytes,2,opt,name=Value,json=value,proto3" json:"Value,omitempty"`
	Entry *hexalog.Entry `protobuf:"bytes,3,opt,name=Entry,json=entry" json:"Entry,omitempty"`
}

func (m *KeyValuePair) Reset()                    { *m = KeyValuePair{} }
func (m *KeyValuePair) String() string            { return proto.CompactTextString(m) }
func (*KeyValuePair) ProtoMessage()               {}
func (*KeyValuePair) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *KeyValuePair) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *KeyValuePair) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *KeyValuePair) GetEntry() *hexalog.Entry {
	if m != nil {
		return m.Entry
	}
	return nil
}

func init() {
	proto.RegisterType((*KeyValuePair)(nil), "fidias.KeyValuePair")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for FidiasRPC service

type FidiasRPCClient interface {
	GetKeyRPC(ctx context.Context, in *KeyValuePair, opts ...grpc.CallOption) (*KeyValuePair, error)
}

type fidiasRPCClient struct {
	cc *grpc.ClientConn
}

func NewFidiasRPCClient(cc *grpc.ClientConn) FidiasRPCClient {
	return &fidiasRPCClient{cc}
}

func (c *fidiasRPCClient) GetKeyRPC(ctx context.Context, in *KeyValuePair, opts ...grpc.CallOption) (*KeyValuePair, error) {
	out := new(KeyValuePair)
	err := grpc.Invoke(ctx, "/fidias.FidiasRPC/GetKeyRPC", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for FidiasRPC service

type FidiasRPCServer interface {
	GetKeyRPC(context.Context, *KeyValuePair) (*KeyValuePair, error)
}

func RegisterFidiasRPCServer(s *grpc.Server, srv FidiasRPCServer) {
	s.RegisterService(&_FidiasRPC_serviceDesc, srv)
}

func _FidiasRPC_GetKeyRPC_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(KeyValuePair)
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
		return srv.(FidiasRPCServer).GetKeyRPC(ctx, req.(*KeyValuePair))
	}
	return interceptor(ctx, in, info, handler)
}

var _FidiasRPC_serviceDesc = grpc.ServiceDesc{
	ServiceName: "fidias.FidiasRPC",
	HandlerType: (*FidiasRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetKeyRPC",
			Handler:    _FidiasRPC_GetKeyRPC_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rpc.proto",
}

func init() { proto.RegisterFile("rpc.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 188 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2c, 0x2a, 0x48, 0xd6,
	0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4b, 0xcb, 0x4c, 0xc9, 0x4c, 0x2c, 0x96, 0x52, 0x4b,
	0xcf, 0x2c, 0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0xcf, 0x48, 0xad, 0x48, 0x4c, 0xca,
	0xc9, 0x4f, 0xce, 0x06, 0xb3, 0x72, 0xf2, 0xd3, 0xf5, 0xe1, 0xea, 0x95, 0x62, 0xb8, 0x78, 0xbc,
	0x53, 0x2b, 0xc3, 0x12, 0x73, 0x4a, 0x53, 0x03, 0x12, 0x33, 0x8b, 0x84, 0x04, 0xb8, 0x98, 0xbd,
	0x53, 0x2b, 0x25, 0x18, 0x15, 0x18, 0x35, 0x78, 0x82, 0x98, 0xb3, 0x53, 0x2b, 0x85, 0x44, 0xb8,
	0x58, 0xc1, 0xd2, 0x12, 0x4c, 0x60, 0x31, 0xd6, 0x32, 0x10, 0x47, 0x48, 0x85, 0x8b, 0xd5, 0x35,
	0xaf, 0xa4, 0xa8, 0x52, 0x82, 0x59, 0x81, 0x51, 0x83, 0xdb, 0x88, 0x4f, 0x0f, 0x6a, 0xb4, 0x1e,
	0x58, 0x34, 0x88, 0x35, 0x15, 0x44, 0x19, 0xb9, 0x71, 0x71, 0xba, 0x81, 0xdd, 0x13, 0x14, 0xe0,
	0x2c, 0x64, 0xc9, 0xc5, 0xe9, 0x9e, 0x5a, 0xe2, 0x9d, 0x5a, 0x09, 0xe2, 0x88, 0xe8, 0x41, 0x1c,
	0xaa, 0x87, 0x6c, 0xbb, 0x14, 0x56, 0x51, 0x25, 0x86, 0x24, 0x36, 0xb0, 0x63, 0x8d, 0x01, 0x01,
	0x00, 0x00, 0xff, 0xff, 0x9c, 0x3e, 0xc7, 0x00, 0xe9, 0x00, 0x00, 0x00,
}
