package p2p

import (
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"net"
)

var _ Protocol = (*ProtocolV10)(nil)

type ProtocolV10 struct {
	net     *Network
	conn    net.Conn
	decoder *gob.Decoder
	encoder *gob.Encoder
	peer    *Peer
}
type V10Msg struct {
	Type    ParcelType
	Crc32   uint32
	Payload []byte
}

func (v10 *ProtocolV10) init(peer *Peer, conn net.Conn, decoder *gob.Decoder, encoder *gob.Encoder) {
	v10.peer = peer
	v10.net = peer.net
	v10.conn = conn
	v10.decoder = decoder
	v10.encoder = encoder
}

func (v10 *ProtocolV10) Send(p *Parcel) error {
	var msg V10Msg
	msg.Type = p.Type
	msg.Crc32 = crc32.Checksum(p.Payload, crcTable)
	msg.Payload = p.Payload
	return v10.encoder.Encode(msg)
}

func (v10 *ProtocolV10) Version() string {
	return "10"
}

func (v10 *ProtocolV10) Receive() (*Parcel, error) {
	var msg V10Msg
	err := v10.decoder.Decode(&msg)
	if err != nil {
		return nil, err
	}

	if len(msg.Payload) == 0 {
		return nil, fmt.Errorf("nul payload")
	}

	csum := crc32.Checksum(msg.Payload, crcTable)
	if csum != msg.Crc32 {
		return nil, fmt.Errorf("invalid checksum")
	}

	p := newParcel(msg.Type, msg.Payload)
	p.Address = v10.conn.RemoteAddr().String()
	return p, nil
}
