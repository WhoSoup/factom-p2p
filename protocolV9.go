package p2p

import (
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"net"
)

var _ Protocol = (*ProtocolV9)(nil)

type ProtocolV9 struct {
	net     *Network
	conn    net.Conn
	decoder *gob.Decoder
	encoder *gob.Encoder
	peer    *Peer
}

func (v9 *ProtocolV9) Init(peer *Peer, conn net.Conn, decoder *gob.Decoder, encoder *gob.Encoder) {
	v9.peer = peer
	v9.net = peer.net
	v9.conn = conn
	v9.decoder = decoder
	v9.encoder = encoder
}

func (v9 *ProtocolV9) Send(p *Parcel) error {
	var msg V9Msg
	msg.Header.Network = v9.net.conf.Network
	msg.Header.Version = 9 // hardcoded
	msg.Header.Type = p.Type
	msg.Header.TargetPeer = p.Address

	msg.Header.NodeID = v9.net.conf.NodeID
	msg.Header.PeerAddress = ""
	msg.Header.PeerPort = v9.net.conf.ListenPort
	msg.Header.AppHash = "NetworkMessage"
	msg.Header.AppType = "Network"

	msg.Payload = p.Payload
	msg.Header.Crc32 = crc32.Checksum(p.Payload, crcTable)
	msg.Header.Length = uint32(len(p.Payload))

	return v9.encoder.Encode(&msg)
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
	p.Payload = msg.Payload
	p.Type = msg.Header.Type
	return p, nil
}
func (v9 *ProtocolV9) Version() string {
	return "9"
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
func (msg V9Msg) Valid() error {
	if msg.Header.Version != 9 {
		return fmt.Errorf("invalid version %v", msg.Header)
	}

	if len(msg.Payload) == 0 {
		return fmt.Errorf("zero-length payload")
	}

	if msg.Header.Length != uint32(len(msg.Payload)) {
		return fmt.Errorf("length in header does not match payload")
	}

	if len(msg.Payload) == 0 {
		return fmt.Errorf("nul payload")
	}

	csum := crc32.Checksum(msg.Payload, crcTable)
	if csum != msg.Header.Crc32 {
		return fmt.Errorf("invalid checksum")
	}

	return nil
}
func (v9 *ProtocolV9) Bootstrap(hs *Handshake) *Parcel {
	p := newParcel(TypePeerRequest, []byte("Peer Request"))
	return p
}
