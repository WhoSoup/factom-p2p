package p2p

import "time"

type Endpoint struct {
	Address  string
	Port     string
	Location uint32

	ConnectionAttempt  time.Time
	ConnectionAttempts uint

	Age          time.Time
	Seed         bool
	IncomingLock time.Duration
}
