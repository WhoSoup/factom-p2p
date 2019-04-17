package p2p

import "time"

type EndpointSettings struct {
	Age         time.Time
	LastAttempt time.Time
	Attempts    int
}
