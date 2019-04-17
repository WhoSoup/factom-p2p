package p2p

import "sync"

type Endpoint struct {
	Address      string
	knownPorts   map[string]bool
	Location     uint32
	Seed         bool
	counterMutex sync.Mutex
	// todo add age, metrics, etc
}
