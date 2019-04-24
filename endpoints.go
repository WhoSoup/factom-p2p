package p2p

import (
	"sync"
	"time"
)

type Endpoints struct {
	mtx  sync.RWMutex
	ends map[IP]endpoint // ip -> bool
	ips  []IP
}

type endpoint struct {
	lock time.Time
	seen time.Time
}

func NewEndpoints() *Endpoints {
	epm := new(Endpoints)
	epm.ends = make(map[IP]endpoint)
	return epm
}

func (epm *Endpoints) Register(ip IP, incoming bool) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if ep, ok := epm.ends[ip]; ok {
		ep.seen = time.Now()
		epm.ends[ip] = ep
	} else {
		epm.ends[ip] = endpoint{seen: time.Now()}
		epm.ips = nil
	}
}
func (epm *Endpoints) Deregister(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	delete(epm.ends, ip)
	epm.ips = nil
}

func (epm *Endpoints) LastSeen(ip IP) time.Time {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return epm.ends[ip].seen
}

func (epm *Endpoints) SetConnectionLock(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if ep, ok := epm.ends[ip]; ok {
		ep.lock = time.Now()
		epm.ends[ip] = ep
	}
}
func (epm *Endpoints) ConnectionLock(ip IP) time.Duration {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return time.Since(epm.ends[ip].lock)
}

func (epm *Endpoints) IPs() []IP {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()

	if epm.ips != nil || len(epm.ends) == 0 {
		return epm.ips
	}

	for ip := range epm.ends {
		epm.ips = append(epm.ips, ip)
	}
	return epm.ips
}
