package p2p

import (
	"sync"
	"time"
)

type EndpointMap struct {
	mtx sync.RWMutex

	net     *Network
	ends    map[string]*Endpoint // address:port -> endpoint
	metrics map[string]bool      // address:nodeid -> metrics
	known   map[string]bool      // address:port -> bool
	lock    map[IP]time.Time     // ip -> time of disconnect
	IPs     []IP

	// TODO implement
	special map[string]bool
	banned  map[string]time.Time
}

func NewEndpointMap(n *Network) *EndpointMap {
	epm := new(EndpointMap)
	epm.net = n
	epm.ends = make(map[string]*Endpoint)
	epm.known = make(map[string]bool)
	epm.lock = make(map[IP]time.Time)
	return epm
}

func (epm *EndpointMap) Register(ip IP, incoming bool) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if !epm.known[ip.String()] {
		epm.known[ip.String()] = true
		epm.IPs = append(epm.IPs, ip)
	}
}

func (epm *EndpointMap) SetConnectionLock(ip IP) {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	epm.lock[ip] = time.Now()
}
func (epm *EndpointMap) ConnectionLocked(ip IP) bool {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return time.Since(epm.lock[ip]) < epm.net.conf.DisconnectLock
}
