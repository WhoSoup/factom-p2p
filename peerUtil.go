package p2p

import (
	"fmt"
	"strconv"
	"time"
)

// PeerMetrics is the data shared to the metrics hook
type PeerMetrics struct {
	Hash             string
	PeerAddress      string
	MomentConnected  time.Time
	PeerQuality      int32
	LastReceive      time.Time
	LastSend         time.Time
	MessagesSent     uint64
	BytesSent        uint64
	MessagesReceived uint64
	BytesReceived    uint64
	Incoming         bool
	PeerType         string
	ConnectionState  string
}

// peerStatus is an indicator for peer manager whether the associated peer is going online or offline
type peerStatus struct {
	peer   *Peer
	online bool
}

type peerParcel struct {
	peer   *Peer
	parcel *Parcel
}

// PeerShare is the data being shared with other peers
type PeerShare struct {
	Address string // Must be in form of x.x.x.x
	Port    string // Must be in form of xxxx
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

// PeerDistance is a sorting metric for sorting peers by location relative to a specific peer
/*
type PeerDistance struct {
	Pivot  uint32
	Sorted []IP
}

func (pd *PeerDistance) Len() int {
	return len(pd.Sorted)
}

func (pd *PeerDistance) Swap(i, j int) {
	pd.Sorted[i], pd.Sorted[j] = pd.Sorted[j], pd.Sorted[i]
}

func (pd *PeerDistance) Less(i, j int) bool {
	return pd.Distance(i) < pd.Distance(j)
}

func (pd *PeerDistance) Distance(i int) uint32 {
	return uintDistance(pd.Pivot, pd.Sorted[i].Location)
}


// uintDistance returns the absolute difference between i and j
func uintDistance(i, j uint32) uint32 {
	if i > j {
		return i - j
	}
	return j - i
}
*/
