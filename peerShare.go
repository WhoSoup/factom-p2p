package p2p

import (
	"fmt"
	"strconv"
	"time"
)

// PeerShare is the data being shared with other peers
type PeerShare struct {
	QualityScore int32     // 0 is neutral quality, negative is a bad peer.
	Address      string    // Must be in form of x.x.x.x
	Port         string    // Must be in form of xxxx
	NodeID       uint64    // a nonce to distinguish multiple nodes behind one IP address
	Hash         string    // This is more of a connection ID than hash right now.
	Location     uint32    // IP address as an int.
	Network      NetworkID // The network this peer reference lives on.
	Type         uint8
	Connections  int                  // Number of successful connections.
	LastContact  time.Time            // Keep track of how long ago we talked to the peer.
	Source       map[string]time.Time // source where we heard from the peer.
}

// String is the concatenation of address and port for easier searching
func (ps PeerShare) String() string {
	return fmt.Sprintf("%s:%s", ps.Address, ps.Port)
}

// Verify checks if the data sent is usable. Does not check if the remote address works
func (ps PeerShare) Verify() bool {
	if _, err := strconv.Atoi(ps.Port); err == nil {
		return ps.Port != "0" && ps.Address != ""
	}
	return false
}
