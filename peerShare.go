package p2p

import "fmt"

// PeerShare is the data being shared with other peers
type PeerShare struct {
	Address      string
	ListenPort   string
	QualityScore int32
}

func (ps PeerShare) ConnectAddress() string {
	return fmt.Sprintf("%s:%s", ps.Address, ps.ListenPort)
}
