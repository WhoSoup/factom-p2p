package p2p

import "net"

// PeerParcel is a temporary hold structure to correlate a parcel with the peer that received it
type PeerParcel struct {
	Peer       *Peer
	Connection *net.Conn
	Parcel     *Parcel
}

type PeerStatus struct {
	Peer   *Peer
	Online bool
}
