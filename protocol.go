package p2p

import (
	"encoding/gob"
	"net"
)

type Protocol interface {
	Init(net *Network, conn net.Conn, read *gob.Decoder, write *gob.Encoder)
	Send(p *Parcel) error
	Receive() (*Parcel, error)
}
