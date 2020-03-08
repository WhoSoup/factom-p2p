// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: protocolV11.proto

package p2p

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type V11Handshake struct {
	Type                 uint32         `protobuf:"varint,1,opt,name=Type,proto3" json:"Type,omitempty"`
	Network              uint32         `protobuf:"varint,2,opt,name=Network,proto3" json:"Network,omitempty"`
	Version              uint32         `protobuf:"varint,3,opt,name=Version,proto3" json:"Version,omitempty"`
	NodeID               uint32         `protobuf:"varint,4,opt,name=NodeID,proto3" json:"NodeID,omitempty"`
	ListenPort           string         `protobuf:"bytes,5,opt,name=ListenPort,proto3" json:"ListenPort,omitempty"`
	Loopback             uint64         `protobuf:"varint,6,opt,name=Loopback,proto3" json:"Loopback,omitempty"`
	Alternatives         []*V11Endpoint `protobuf:"bytes,7,rep,name=Alternatives,proto3" json:"Alternatives,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *V11Handshake) Reset()         { *m = V11Handshake{} }
func (m *V11Handshake) String() string { return proto.CompactTextString(m) }
func (*V11Handshake) ProtoMessage()    {}
func (*V11Handshake) Descriptor() ([]byte, []int) {
	return fileDescriptor_88431f7e68323b26, []int{0}
}
func (m *V11Handshake) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *V11Handshake) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_V11Handshake.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *V11Handshake) XXX_Merge(src proto.Message) {
	xxx_messageInfo_V11Handshake.Merge(m, src)
}
func (m *V11Handshake) XXX_Size() int {
	return m.Size()
}
func (m *V11Handshake) XXX_DiscardUnknown() {
	xxx_messageInfo_V11Handshake.DiscardUnknown(m)
}

var xxx_messageInfo_V11Handshake proto.InternalMessageInfo

func (m *V11Handshake) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *V11Handshake) GetNetwork() uint32 {
	if m != nil {
		return m.Network
	}
	return 0
}

func (m *V11Handshake) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *V11Handshake) GetNodeID() uint32 {
	if m != nil {
		return m.NodeID
	}
	return 0
}

func (m *V11Handshake) GetListenPort() string {
	if m != nil {
		return m.ListenPort
	}
	return ""
}

func (m *V11Handshake) GetLoopback() uint64 {
	if m != nil {
		return m.Loopback
	}
	return 0
}

func (m *V11Handshake) GetAlternatives() []*V11Endpoint {
	if m != nil {
		return m.Alternatives
	}
	return nil
}

type V11Endpoint struct {
	Host                 string   `protobuf:"bytes,1,opt,name=Host,proto3" json:"Host,omitempty"`
	Port                 string   `protobuf:"bytes,2,opt,name=Port,proto3" json:"Port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *V11Endpoint) Reset()         { *m = V11Endpoint{} }
func (m *V11Endpoint) String() string { return proto.CompactTextString(m) }
func (*V11Endpoint) ProtoMessage()    {}
func (*V11Endpoint) Descriptor() ([]byte, []int) {
	return fileDescriptor_88431f7e68323b26, []int{1}
}
func (m *V11Endpoint) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *V11Endpoint) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_V11Endpoint.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *V11Endpoint) XXX_Merge(src proto.Message) {
	xxx_messageInfo_V11Endpoint.Merge(m, src)
}
func (m *V11Endpoint) XXX_Size() int {
	return m.Size()
}
func (m *V11Endpoint) XXX_DiscardUnknown() {
	xxx_messageInfo_V11Endpoint.DiscardUnknown(m)
}

var xxx_messageInfo_V11Endpoint proto.InternalMessageInfo

func (m *V11Endpoint) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *V11Endpoint) GetPort() string {
	if m != nil {
		return m.Port
	}
	return ""
}

type V11Msg struct {
	Type                 uint32   `protobuf:"varint,1,opt,name=Type,proto3" json:"Type,omitempty"`
	Payload              []byte   `protobuf:"bytes,2,opt,name=Payload,proto3" json:"Payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *V11Msg) Reset()         { *m = V11Msg{} }
func (m *V11Msg) String() string { return proto.CompactTextString(m) }
func (*V11Msg) ProtoMessage()    {}
func (*V11Msg) Descriptor() ([]byte, []int) {
	return fileDescriptor_88431f7e68323b26, []int{2}
}
func (m *V11Msg) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *V11Msg) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_V11Msg.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *V11Msg) XXX_Merge(src proto.Message) {
	xxx_messageInfo_V11Msg.Merge(m, src)
}
func (m *V11Msg) XXX_Size() int {
	return m.Size()
}
func (m *V11Msg) XXX_DiscardUnknown() {
	xxx_messageInfo_V11Msg.DiscardUnknown(m)
}

var xxx_messageInfo_V11Msg proto.InternalMessageInfo

func (m *V11Msg) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *V11Msg) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type V11Share struct {
	Share                []*V11Endpoint `protobuf:"bytes,1,rep,name=Share,proto3" json:"Share,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *V11Share) Reset()         { *m = V11Share{} }
func (m *V11Share) String() string { return proto.CompactTextString(m) }
func (*V11Share) ProtoMessage()    {}
func (*V11Share) Descriptor() ([]byte, []int) {
	return fileDescriptor_88431f7e68323b26, []int{3}
}
func (m *V11Share) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *V11Share) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_V11Share.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *V11Share) XXX_Merge(src proto.Message) {
	xxx_messageInfo_V11Share.Merge(m, src)
}
func (m *V11Share) XXX_Size() int {
	return m.Size()
}
func (m *V11Share) XXX_DiscardUnknown() {
	xxx_messageInfo_V11Share.DiscardUnknown(m)
}

var xxx_messageInfo_V11Share proto.InternalMessageInfo

func (m *V11Share) GetShare() []*V11Endpoint {
	if m != nil {
		return m.Share
	}
	return nil
}

func init() {
	proto.RegisterType((*V11Handshake)(nil), "p2p.V11Handshake")
	proto.RegisterType((*V11Endpoint)(nil), "p2p.V11Endpoint")
	proto.RegisterType((*V11Msg)(nil), "p2p.V11Msg")
	proto.RegisterType((*V11Share)(nil), "p2p.V11Share")
}

func init() { proto.RegisterFile("protocolV11.proto", fileDescriptor_88431f7e68323b26) }

var fileDescriptor_88431f7e68323b26 = []byte{
	// 303 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x91, 0xcb, 0x4a, 0x03, 0x31,
	0x14, 0x86, 0x4d, 0x2f, 0xd3, 0xf6, 0xb4, 0x42, 0xcd, 0x42, 0x82, 0x8b, 0x61, 0x98, 0x85, 0xcc,
	0xaa, 0x90, 0xf1, 0xb2, 0x57, 0x14, 0x2a, 0xd4, 0x52, 0x46, 0x99, 0x7d, 0xda, 0x09, 0x76, 0x68,
	0xc9, 0x09, 0x49, 0x50, 0xfa, 0x26, 0x3e, 0x92, 0x4b, 0x1f, 0xa1, 0xd4, 0x17, 0x91, 0x49, 0x5b,
	0xa9, 0x8b, 0xee, 0xfe, 0xef, 0xff, 0x12, 0x38, 0x27, 0x81, 0x33, 0x6d, 0xd0, 0xe1, 0x0c, 0x97,
	0x39, 0xe7, 0x03, 0x9f, 0x69, 0x5d, 0xa7, 0x3a, 0x5e, 0x13, 0xe8, 0xe5, 0x9c, 0x0f, 0x85, 0x2a,
	0xec, 0x5c, 0x2c, 0x24, 0xa5, 0xd0, 0x78, 0x5d, 0x69, 0xc9, 0x48, 0x44, 0x92, 0xd3, 0xcc, 0x67,
	0xca, 0xa0, 0x35, 0x96, 0xee, 0x03, 0xcd, 0x82, 0xd5, 0x7c, 0xbd, 0xc7, 0xca, 0xe4, 0xd2, 0xd8,
	0x12, 0x15, 0xab, 0x6f, 0xcd, 0x0e, 0xe9, 0x39, 0x04, 0x63, 0x2c, 0xe4, 0xd3, 0x03, 0x6b, 0x78,
	0xb1, 0x23, 0x1a, 0x02, 0x8c, 0x4a, 0xeb, 0xa4, 0x9a, 0xa0, 0x71, 0xac, 0x19, 0x91, 0xa4, 0x93,
	0x1d, 0x34, 0xf4, 0x02, 0xda, 0x23, 0x44, 0x3d, 0x15, 0xb3, 0x05, 0x0b, 0x22, 0x92, 0x34, 0xb2,
	0x3f, 0xa6, 0xd7, 0xd0, 0xbb, 0x5b, 0x3a, 0x69, 0x94, 0x70, 0xe5, 0xbb, 0xb4, 0xac, 0x15, 0xd5,
	0x93, 0x6e, 0xda, 0x1f, 0xe8, 0x54, 0x0f, 0x72, 0xce, 0x1f, 0x55, 0xa1, 0xb1, 0x54, 0x2e, 0xfb,
	0x77, 0x2a, 0xbe, 0x81, 0xee, 0x81, 0xac, 0x16, 0x1c, 0xa2, 0x75, 0x7e, 0xc1, 0x4e, 0xe6, 0x73,
	0xd5, 0xf9, 0x71, 0x6a, 0xdb, 0xae, 0xca, 0xf1, 0x2d, 0x04, 0x39, 0xe7, 0xcf, 0xf6, 0xed, 0xd8,
	0x93, 0x4c, 0xc4, 0x6a, 0x89, 0xa2, 0xf0, 0x97, 0x7a, 0xd9, 0x1e, 0xe3, 0x14, 0xda, 0x39, 0xe7,
	0x2f, 0x73, 0x61, 0x24, 0xbd, 0x84, 0xa6, 0x0f, 0x8c, 0x1c, 0x99, 0x74, 0xab, 0xef, 0xfb, 0x5f,
	0x9b, 0x90, 0x7c, 0x6f, 0x42, 0xb2, 0xde, 0x84, 0xe4, 0xf3, 0x27, 0x3c, 0x99, 0x06, 0xfe, 0x8f,
	0xae, 0x7e, 0x03, 0x00, 0x00, 0xff, 0xff, 0x02, 0x9b, 0xc9, 0x29, 0xb8, 0x01, 0x00, 0x00,
}

func (m *V11Handshake) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *V11Handshake) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *V11Handshake) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Alternatives) > 0 {
		for iNdEx := len(m.Alternatives) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Alternatives[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintProtocolV11(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x3a
		}
	}
	if m.Loopback != 0 {
		i = encodeVarintProtocolV11(dAtA, i, uint64(m.Loopback))
		i--
		dAtA[i] = 0x30
	}
	if len(m.ListenPort) > 0 {
		i -= len(m.ListenPort)
		copy(dAtA[i:], m.ListenPort)
		i = encodeVarintProtocolV11(dAtA, i, uint64(len(m.ListenPort)))
		i--
		dAtA[i] = 0x2a
	}
	if m.NodeID != 0 {
		i = encodeVarintProtocolV11(dAtA, i, uint64(m.NodeID))
		i--
		dAtA[i] = 0x20
	}
	if m.Version != 0 {
		i = encodeVarintProtocolV11(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x18
	}
	if m.Network != 0 {
		i = encodeVarintProtocolV11(dAtA, i, uint64(m.Network))
		i--
		dAtA[i] = 0x10
	}
	if m.Type != 0 {
		i = encodeVarintProtocolV11(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *V11Endpoint) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *V11Endpoint) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *V11Endpoint) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Port) > 0 {
		i -= len(m.Port)
		copy(dAtA[i:], m.Port)
		i = encodeVarintProtocolV11(dAtA, i, uint64(len(m.Port)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Host) > 0 {
		i -= len(m.Host)
		copy(dAtA[i:], m.Host)
		i = encodeVarintProtocolV11(dAtA, i, uint64(len(m.Host)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *V11Msg) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *V11Msg) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *V11Msg) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Payload) > 0 {
		i -= len(m.Payload)
		copy(dAtA[i:], m.Payload)
		i = encodeVarintProtocolV11(dAtA, i, uint64(len(m.Payload)))
		i--
		dAtA[i] = 0x12
	}
	if m.Type != 0 {
		i = encodeVarintProtocolV11(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *V11Share) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *V11Share) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *V11Share) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Share) > 0 {
		for iNdEx := len(m.Share) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Share[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintProtocolV11(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintProtocolV11(dAtA []byte, offset int, v uint64) int {
	offset -= sovProtocolV11(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *V11Handshake) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != 0 {
		n += 1 + sovProtocolV11(uint64(m.Type))
	}
	if m.Network != 0 {
		n += 1 + sovProtocolV11(uint64(m.Network))
	}
	if m.Version != 0 {
		n += 1 + sovProtocolV11(uint64(m.Version))
	}
	if m.NodeID != 0 {
		n += 1 + sovProtocolV11(uint64(m.NodeID))
	}
	l = len(m.ListenPort)
	if l > 0 {
		n += 1 + l + sovProtocolV11(uint64(l))
	}
	if m.Loopback != 0 {
		n += 1 + sovProtocolV11(uint64(m.Loopback))
	}
	if len(m.Alternatives) > 0 {
		for _, e := range m.Alternatives {
			l = e.Size()
			n += 1 + l + sovProtocolV11(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *V11Endpoint) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Host)
	if l > 0 {
		n += 1 + l + sovProtocolV11(uint64(l))
	}
	l = len(m.Port)
	if l > 0 {
		n += 1 + l + sovProtocolV11(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *V11Msg) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Type != 0 {
		n += 1 + sovProtocolV11(uint64(m.Type))
	}
	l = len(m.Payload)
	if l > 0 {
		n += 1 + l + sovProtocolV11(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *V11Share) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Share) > 0 {
		for _, e := range m.Share {
			l = e.Size()
			n += 1 + l + sovProtocolV11(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovProtocolV11(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozProtocolV11(x uint64) (n int) {
	return sovProtocolV11(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *V11Handshake) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtocolV11
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: V11Handshake: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: V11Handshake: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Network", wireType)
			}
			m.Network = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Network |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NodeID", wireType)
			}
			m.NodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NodeID |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ListenPort", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthProtocolV11
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ListenPort = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Loopback", wireType)
			}
			m.Loopback = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Loopback |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Alternatives", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProtocolV11
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Alternatives = append(m.Alternatives, &V11Endpoint{})
			if err := m.Alternatives[len(m.Alternatives)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtocolV11(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *V11Endpoint) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtocolV11
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: V11Endpoint: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: V11Endpoint: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Host", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthProtocolV11
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Host = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Port", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthProtocolV11
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Port = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtocolV11(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *V11Msg) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtocolV11
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: V11Msg: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: V11Msg: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthProtocolV11
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Payload = append(m.Payload[:0], dAtA[iNdEx:postIndex]...)
			if m.Payload == nil {
				m.Payload = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtocolV11(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *V11Share) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtocolV11
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: V11Share: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: V11Share: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Share", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProtocolV11
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Share = append(m.Share, &V11Endpoint{})
			if err := m.Share[len(m.Share)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtocolV11(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthProtocolV11
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipProtocolV11(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProtocolV11
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowProtocolV11
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthProtocolV11
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupProtocolV11
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthProtocolV11
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthProtocolV11        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProtocolV11          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupProtocolV11 = fmt.Errorf("proto: unexpected end of group")
)