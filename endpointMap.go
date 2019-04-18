package p2p

import (
	"sync"
	"time"
)

type EndpointMap struct {
	mtx sync.RWMutex

	net      *Network
	ends     map[string]*Endpoint // address:port -> endpoint
	metrics  map[string]bool      // address:nodeid -> metrics
	known    map[string]bool      // address:port -> bool
	incoming map[string]bool      // address:port -> bool
	IPs      []IP

	// TODO implement
	special map[string]bool
	banned  map[string]time.Time
}

func NewEndpointMap(n *Network) *EndpointMap {
	epm := new(EndpointMap)
	epm.net = n
	epm.ends = make(map[string]*Endpoint)
	epm.known = make(map[string]bool)
	epm.incoming = make(map[string]bool)
	return epm
}

func (epm *EndpointMap) Register(ip IP, incoming bool) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if !epm.known[ip.String()] {
		epm.known[ip.String()] = true
		epm.IPs = append(epm.IPs, ip)
	}
	epm.incoming[ip.String()] = epm.incoming[ip.String()] || incoming
}

func (epm *EndpointMap) IsIncoming(ip IP) bool {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return epm.incoming[ip.String()]
}
