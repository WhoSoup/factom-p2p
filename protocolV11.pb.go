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

func init() {
	proto.RegisterType((*V11Handshake)(nil), "p2p.V11Handshake")
	proto.RegisterType((*V11Endpoint)(nil), "p2p.V11Endpoint")
	proto.RegisterType((*V11Msg)(nil), "p2p.V11Msg")
}

func init() { proto.RegisterFile("protocolV11.proto", fileDescriptor_88431f7e68323b26) }

var fileDescriptor_88431f7e68323b26 = []byte{
	// 283 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x90, 0x4f, 0x4b, 0xc3, 0x30,
	0x18, 0xc6, 0xcd, 0x5a, 0x3b, 0x97, 0x55, 0x98, 0x39, 0x48, 0xf0, 0x50, 0x4a, 0x4f, 0x3d, 0x15,
	0x52, 0xff, 0xdc, 0x15, 0x85, 0x09, 0x73, 0x8c, 0x20, 0xbd, 0x67, 0x6b, 0xd0, 0xd2, 0x92, 0x37,
	0x34, 0x41, 0xd9, 0x37, 0xf1, 0x23, 0x79, 0xf4, 0x23, 0x8c, 0xfa, 0x45, 0xa4, 0xd9, 0x26, 0xf3,
	0xe0, 0xed, 0xf9, 0x3d, 0xbf, 0x04, 0xde, 0xf7, 0xc5, 0x67, 0xba, 0x05, 0x0b, 0x2b, 0x68, 0x0a,
	0xc6, 0x32, 0x97, 0x89, 0xa7, 0x73, 0x9d, 0x6c, 0x10, 0x0e, 0x0b, 0xc6, 0xa6, 0x42, 0x95, 0xe6,
	0x55, 0xd4, 0x92, 0x10, 0xec, 0x3f, 0xaf, 0xb5, 0xa4, 0x28, 0x46, 0xe9, 0x29, 0x77, 0x99, 0x50,
	0x3c, 0x9c, 0x4b, 0xfb, 0x0e, 0x6d, 0x4d, 0x07, 0xae, 0xde, 0x63, 0x6f, 0x0a, 0xd9, 0x9a, 0x0a,
	0x14, 0xf5, 0xb6, 0x66, 0x87, 0xe4, 0x1c, 0x07, 0x73, 0x28, 0xe5, 0xe3, 0x3d, 0xf5, 0x9d, 0xd8,
	0x11, 0x89, 0x30, 0x9e, 0x55, 0xc6, 0x4a, 0xb5, 0x80, 0xd6, 0xd2, 0xe3, 0x18, 0xa5, 0x23, 0x7e,
	0xd0, 0x90, 0x0b, 0x7c, 0x32, 0x03, 0xd0, 0x4b, 0xb1, 0xaa, 0x69, 0x10, 0xa3, 0xd4, 0xe7, 0xbf,
	0x4c, 0xae, 0x70, 0x78, 0xdb, 0x58, 0xd9, 0x2a, 0x61, 0xab, 0x37, 0x69, 0xe8, 0x30, 0xf6, 0xd2,
	0x71, 0x3e, 0xc9, 0x74, 0xae, 0xb3, 0x82, 0xb1, 0x07, 0x55, 0x6a, 0xa8, 0x94, 0xe5, 0x7f, 0x5e,
	0x25, 0xd7, 0x78, 0x7c, 0x20, 0xfb, 0x05, 0xa7, 0x60, 0xac, 0x5b, 0x70, 0xc4, 0x5d, 0xee, 0x3b,
	0x37, 0xce, 0x60, 0xdb, 0xf5, 0x39, 0xb9, 0xc1, 0x41, 0xc1, 0xd8, 0x93, 0x79, 0xf9, 0xef, 0x24,
	0x0b, 0xb1, 0x6e, 0x40, 0x94, 0xee, 0x53, 0xc8, 0xf7, 0x78, 0x37, 0xf9, 0xec, 0x22, 0xf4, 0xd5,
	0x45, 0x68, 0xd3, 0x45, 0xe8, 0xe3, 0x3b, 0x3a, 0x5a, 0x06, 0xee, 0xde, 0x97, 0x3f, 0x01, 0x00,
	0x00, 0xff, 0xff, 0x0d, 0xc7, 0x04, 0x64, 0x84, 0x01, 0x00, 0x00,
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
