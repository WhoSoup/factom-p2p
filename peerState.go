package p2p

import "fmt"

type PeerState struct {
	Peer  *Peer
	State PeerStateType
}

func (ps PeerState) String() string {
	return fmt.Sprintf("%s - %s", ps.Peer, ps.State)
}

// PeerState is the states for the Peer's state machine
type PeerStateType uint8

// The peer state machine's states
const (
	Offline PeerStateType = iota
	Connecting
	Online
)

func (ps PeerStateType) String() string {
	switch ps {
	case Offline:
		return "Offline"
	case Connecting:
		return "Connecting"
	case Online:
		return "Online"
	default:
		return "Unknown state"
	}
}
