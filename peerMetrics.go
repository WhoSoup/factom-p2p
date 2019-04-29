package p2p

import "time"

type PeerMetrics struct {
	Hash             string
	PeerAddress      string
	MomentConnected  time.Time
	PeerQuality      int
	LastReceive      time.Time
	LastSend         time.Time
	MessagesSent     uint64
	BytesSent        uint64
	MessagesReceived uint64
	BytesReceived    uint64
	Incoming         bool
}
