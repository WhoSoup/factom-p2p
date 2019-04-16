package p2p

import "sync"

type Endpoint struct {
	Address       string
	Location      uint32
	knownPorts    map[string]bool
	TotalIncoming int
	TotalOutgoing int
	Seed          bool
	counterMutex  sync.Mutex

	// todo add age, metrics, etc
}

func NewEndpoint(addr string) (*Endpoint, error) {
	e := new(Endpoint)
	e.Address = addr
	e.knownPorts = make(map[string]bool)
	if loc, err := IP2Location(addr); err == nil {
		e.Location = loc
	} else {
		return nil, err
	}

	return e, nil
}

func (e *Endpoint) Connections() int {
	return e.TotalIncoming + e.TotalOutgoing
}

func (e *Endpoint) Register(p *Peer) {
	e.counterMutex.Lock()
	defer e.counterMutex.Unlock()
	if p.IsIncoming {
		e.TotalIncoming++
	} else {
		e.TotalOutgoing++
	}
}
func (e *Endpoint) Unregister(p *Peer) {
	e.counterMutex.Lock()
	defer e.counterMutex.Unlock()
	if p.IsIncoming {
		e.TotalIncoming--
	} else {
		e.TotalOutgoing--
	}
}
