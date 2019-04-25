package p2p

import (
	"encoding/json"
	"sync"
	"time"
)

type Endpoints struct {
	Ends map[string]Endpoint  `json:"endpoints"`
	Bans map[string]time.Time `json:"bans"`
	mtx  sync.RWMutex
	ips  []IP
}

type Endpoint struct {
	IP     IP                   `json:"ip"`
	Seen   time.Time            `json:"seen"`
	Source map[string]time.Time `json:"source"`
	lock   time.Time
}

func NewEndpoints() *Endpoints {
	epm := new(Endpoints)
	epm.Ends = make(map[string]Endpoint)
	epm.Bans = make(map[string]time.Time)
	return epm
}

func (epm *Endpoints) Register(ip IP, incoming bool, source string) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	ep := epm.Ends[ip.String()]
	if ep.Source == nil {
		ep.Source = make(map[string]time.Time)
	}
	ep.Seen = time.Now()
	ep.Source[source] = time.Now()
	ep.IP = ip
	epm.Ends[ip.String()] = ep
	epm.ips = nil
}

func (epm *Endpoints) Refresh(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if ep, ok := epm.Ends[ip.String()]; ok {
		ep.Seen = time.Now()
		epm.Ends[ip.String()] = ep
	}
}

func (epm *Endpoints) Deregister(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	delete(epm.Ends, ip.String())
	epm.ips = nil
}

func (epm *Endpoints) Ban(addr string, t time.Time) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	for ip := range epm.Ends {
		if ip == addr {
			delete(epm.Ends, ip)
		}
	}
	epm.Bans[addr] = t
	epm.ips = nil
}

func (epm *Endpoints) Banned(addr string) bool {
	return time.Now().Before(epm.Bans[addr])
}

func (epm *Endpoints) LastSeen(ip IP) time.Time {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return epm.Ends[ip.String()].Seen
}

func (epm *Endpoints) SetConnectionLock(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if ep, ok := epm.Ends[ip.String()]; ok {
		ep.lock = time.Now()
		epm.Ends[ip.String()] = ep
	}
}
func (epm *Endpoints) ConnectionLock(ip IP) time.Duration {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return time.Since(epm.Ends[ip.String()].lock)
}

func (epm *Endpoints) IPs() []IP {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()

	if epm.ips != nil || len(epm.Ends) == 0 {
		return epm.ips
	}

	for _, ep := range epm.Ends {
		epm.ips = append(epm.ips, ep.IP)
	}
	return epm.ips
}

func (epm *Endpoints) cleanup(cutoff time.Duration) uint {
	removed := uint(0)
	for addr, ep := range epm.Ends {
		if time.Since(ep.Seen) > cutoff {
			delete(epm.Ends, addr)
			removed++
		}
	}
	for addr, ban := range epm.Bans {
		if ban.Before(time.Now()) {
			delete(epm.Bans, addr)
		}
	}
	epm.ips = nil
	return removed
}

func (epm *Endpoints) Persist(cutoff time.Duration) ([]byte, error) {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	epm.cleanup(cutoff)
	return json.Marshal(epm)
}
