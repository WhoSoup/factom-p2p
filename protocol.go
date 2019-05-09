package p2p

import "github.com/whosoup/factom-p2p/util"

// Protocol is the interface for reading and writing parcels to
// the underlying connection. The job of a protocol is to encode a Parcel
// and send it over TCP to another instance of the same Protocol on the
// other end
//
// Send:    Parcel => Protocol Encoder => Protocol Format => TCP
// Receive: TCP => Protocol Format => Protocol Decoder => Parcel
//
// Every protocol should be bootstraped in peer:bootstrapProtocol() where
// it can be initialized with the required serialization methods
type Protocol interface {
	Send(p *Parcel) error
	Receive() (*Parcel, error)
	MakePeerShare(ps []util.IP) ([]byte, error)
	ParsePeerShare(payload []byte) ([]PeerShare, error)
	Version() string
}
