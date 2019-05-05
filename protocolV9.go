package p2p

import (
	"encoding/gob"
	"fmt"
	"net"
)

var _ Protocol = (*ProtocolV9)(nil)

type ProtocolV9 struct {
	net     *Network
	conn    net.Conn
	decoder *gob.Decoder
	encoder *gob.Encoder
}

func (v9 *ProtocolV9) Init(net *Network, conn net.Conn, decoder *gob.Decoder, encoder *gob.Encoder) {
	v9.net = net
	v9.conn = conn
	v9.decoder = decoder
	v9.encoder = encoder
}

func (v9 *ProtocolV9) Send(p *Parcel) error {
	msg := new(V9Msg)
	msg.Header.NodeID = v9.net.conf.NodeID
	msg.Header.Version = v9.net.conf.ProtocolVersion
	msg.Header.Network = v9.net.conf.Network
	msg.Header.PeerPort = v9.net.conf.ListenPort
	msg.Payload = p.Payload
	msg.Header.Crc32 = p.Crc32
	msg.Header.Length = uint32(len(p.Payload))
	msg.Header.AppHash = p.AppHash
	msg.Header.AppType = p.AppType

	return v9.encoder.Encode(msg)
}

func (v9 *ProtocolV9) Receive() (*Parcel, error) {
	var msg V9Msg
	err := v9.decoder.Decode(&msg)
	if err != nil {
		return nil, err
	}

	if err = msg.Valid(); err != nil {
		return nil, err
	}

	p := new(Parcel)
	p.Address = msg.Header.TargetPeer
	p.AppHash = msg.Header.AppHash
	p.AppType = msg.Header.AppType
	p.Crc32 = msg.Header.Crc32
	p.Payload = msg.Payload
	p.Type = msg.Header.Type
	return p, nil
}

type V9Msg struct {
	Header  V9Header
	Payload []byte
}

type V9Header struct {
	Network     NetworkID  // 4 bytes - the network we are on (eg testnet, main net, etc.)
	Version     uint16     // 2 bytes - the version of the protocol we are running.
	Type        ParcelType // 2 bytes - network level commands (eg: ping/pong)
	Length      uint32     // 4 bytes - length of the payload (that follows this header) in bytes
	TargetPeer  string     // ? bytes - "" or nil for broadcast, otherwise the destination peer's hash.
	Crc32       uint32     // 4 bytes - data integrity hash (of the payload itself.)
	_           uint16     // 2 bytes - in case of multipart parcels, indicates which part this corresponds to, otherwise should be 0
	_           uint16     // 2 bytes - in case of multipart parcels, indicates the total number of parts that the receiver should expect
	NodeID      uint64
	PeerAddress string // address of the peer set by connection to know who sent message (for tracking source of other peers)
	PeerPort    string // port of the peer , or we are listening on
	AppHash     string // Application specific message hash, for tracing
	AppType     string // Application specific message type, for tracing
}

// Valid checks header for inconsistencies
func (msg *V9Msg) Valid() error {
	if msg == nil {
		return fmt.Errorf("nil parcel")
	}

	head := msg.Header
	if head.Version != 9 {
		return fmt.Errorf("invalid version")
	}

	if len(msg.Payload) == 0 {
		return fmt.Errorf("zero-length payload")
	}

	if head.Length != uint32(len(msg.Payload)) {
		return fmt.Errorf("length in header does not match payload")
	}

	return nil
}
