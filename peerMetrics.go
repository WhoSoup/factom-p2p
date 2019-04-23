package p2p

import "time"

type PeerMetrics struct {
	Connected time.Time
	Quality   int

	LastReceive time.Time
	LastSend    time.Time

	ParcelsSent uint64
	BytesSent   uint64

	ParcelsReceived uint64
	BytesReceived   uint64

	Incoming bool
}
