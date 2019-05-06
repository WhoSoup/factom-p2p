package p2p

import (
	"encoding/gob"
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

func (v10 *ProtocolV10) Init(peer *Peer, conn net.Conn, decoder *gob.Decoder, encoder *gob.Encoder) {
	v10.peer = peer
	v10.net = peer.net
	v10.conn = conn
	v10.decoder = decoder
	v10.encoder = encoder
}

func (v10 *ProtocolV10) Send(p *Parcel) error {
	msg := V10Msg(*p)
	return v10.encoder.Encode(&msg)
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

	p := Parcel(msg)
	return &p, nil
}

type V10Msg Parcel
