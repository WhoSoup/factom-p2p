package p2p

import (
	"fmt"
	"strconv"
)

// PeerShare is the data being shared with other peers
type PeerShare struct {
	Address      string
	Port         string
	QualityScore int32
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
