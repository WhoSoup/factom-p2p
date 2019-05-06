package p2p

type Protocol interface {
	Send(p *Parcel) error
	Receive() (*Parcel, error)
	Version() string
	Bootstrap(hs *Handshake) *Parcel
}
