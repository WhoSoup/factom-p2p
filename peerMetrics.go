package p2p

import "time"

type PeerMetrics struct {
	Age     time.Time
	Quality int

	Connected    time.Time
	Disconnected time.Time

	ConnectionAttempt  time.Time
	ConnectionAttempts uint

	LastRead time.Time
	LastSend time.Time

	ParcelsSent uint64
	BytesSent   uint64

	ParcelsReceived uint64
	BytesReceived   uint64
}
