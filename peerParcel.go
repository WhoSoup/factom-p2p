package p2p

// PeerParcel is a temporary hold structure to correlate a parcel with the peer that received it
type PeerParcel struct {
	Peer   *Peer
	Parcel *Parcel
}

type PeerStatus struct {
	Peer   *Peer
	Online bool
}
