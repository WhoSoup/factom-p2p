package util

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Endpoints is a collection of known ip addresses in the network.
// Aka the partial peer view.
// Endpoints are unique and there can be only one for a given IP
type Endpoints struct {
	Ends    map[string]endpoint  `json:"endpoints"` // addr:port -> endpoint
	Bans    map[string]time.Time `json:"bans"`
	special map[string]bool      // (address|address:port) -> bool
	mtx     sync.RWMutex
	ips     []IP
}

type endpoint struct {
	IP           IP                   `json:"ip"`
	Seen         time.Time            `json:"seen"`
	Source       map[string]time.Time `json:"source"`
	Connected    time.Time            `json:"connected"`
	Disconnected time.Time            `json:"disconnected"`
	connections  uint
	lock         time.Time
}

// NewEndpoints creates an empty endpoint holder
func NewEndpoints() *Endpoints {
	epm := new(Endpoints)
	epm.Ends = make(map[string]endpoint)
	epm.Bans = make(map[string]time.Time)
	epm.special = make(map[string]bool)
	return epm
}

// Total returns the total amount of connected peers
func (epm *Endpoints) Total() int {
	return len(epm.Ends)
}

// Register an IP in the system along with the source of where it's from
func (epm *Endpoints) Register(ip IP, source string) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	ep := epm.Ends[ip.String()]
	if ep.Source == nil {
		ep.Source = make(map[string]time.Time)
	}
	// refresh only if we haven't seen them from this source before
	if ep.Source[source].IsZero() {
		ep.Seen = time.Now()
		ep.Source[source] = time.Now()
	}
	if source == "Special" {
		epm.special[ip.String()] = true
		epm.special[ip.Address] = true
	}
	ep.IP = ip
	epm.Ends[ip.String()] = ep
	epm.ips = nil
}

// AddConnection registers that a connection to this endpoint has been established
func (epm *Endpoints) AddConnection(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	ep := epm.Ends[ip.String()]
	ep.connections++
	if ep.Connected.IsZero() {
		ep.Connected = time.Now()
	}
	ep.Disconnected = time.Time{}
	epm.Ends[ip.String()] = ep
}

// RemoveConnection registers that a connection to this endpoint has been severed
func (epm *Endpoints) RemoveConnection(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	ep := epm.Ends[ip.String()]
	if ep.connections > 0 {
		ep.connections--
		if ep.connections == 0 {
			ep.Disconnected = time.Now()
		}
		epm.Ends[ip.String()] = ep
	}
}

// Connections returns the amount of connections to this endpoint along with a duration of
// how long at least one connection has been active
func (epm *Endpoints) Connections(ip IP) (uint, time.Duration) {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	ep := epm.Ends[ip.String()]
	t := ep.Disconnected
	if t.IsZero() {
		t = time.Now()
	}
	if ep.Connected.IsZero() {
		return ep.connections, 0
	}
	return ep.connections, t.Sub(ep.Connected)
}

// Deregister removes an endpoint from the store
func (epm *Endpoints) Deregister(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	delete(epm.Ends, ip.String())
	epm.ips = nil
}

// Ban all endpoints with a given ip address until a certain time
func (epm *Endpoints) Ban(addr string, t time.Time) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	for i, ep := range epm.Ends {
		if ep.IP.Address == addr {
			delete(epm.Ends, i)
		}
	}
	epm.Bans[addr] = t
	epm.ips = nil
}

// Banned checks if an ip address is banned
func (epm *Endpoints) Banned(addr string) bool {
	return time.Now().Before(epm.Bans[addr])
}

// LastSeen returns the time of the last activity
func (epm *Endpoints) LastSeen(ip IP) time.Time {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return epm.Ends[ip.String()].Seen
}

// Lock an endpoint for a specific duration
func (epm *Endpoints) Lock(ip IP, dur time.Duration) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if ep, ok := epm.Ends[ip.String()]; ok {
		ep.lock = time.Now().Add(dur)
		epm.Ends[ip.String()] = ep
	}
}

// Unlock an endpoint again
func (epm *Endpoints) Unlock(ip IP) {
	epm.mtx.Lock()
	defer epm.mtx.Unlock()
	if ep, ok := epm.Ends[ip.String()]; ok {
		ep.lock = time.Time{}
		epm.Ends[ip.String()] = ep
	}
}

// IsLocked checks if an endpoint is locked
func (epm *Endpoints) IsLocked(ip IP) bool {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return time.Now().Before(epm.Ends[ip.String()].lock)
}

// IsSpecial checks if the given endpoint is special
func (epm *Endpoints) IsSpecial(ip IP) bool {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return epm.special[ip.String()]
}

// IsSpecialAddress checks if there is a special endpoint on this ip address
func (epm *Endpoints) IsSpecialAddress(addr string) bool {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	return epm.special[addr]
}

// IPs returns a concurrency safe slice of the current endpoints to loop over.
// The slice should not be modified.
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

// Cleanup performs janitorial duties on the data, removing endpoints that have
// not been connected to since the cutoff and removing expired bans
func (epm *Endpoints) Cleanup(cutoff time.Duration) uint {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	removed := uint(0)
	for addr, ep := range epm.Ends {
		if time.Since(ep.Connected) > cutoff {
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

// trim the endpoints for persisting according to the level. see configuration for details
func (epm *Endpoints) trim(level uint, min time.Duration, cutoff time.Duration) *Endpoints {
	epm.mtx.RLock()
	defer epm.mtx.RUnlock()
	e := NewEndpoints()
	e.Bans = epm.Bans
	cut := time.Now().Add(-cutoff)
	for ip, end := range epm.Ends {
		// cutoff
		if !end.Disconnected.IsZero() && end.Disconnected.Before(cut) {
			continue
		}
		if level >= 1 && end.timeConnected() < min {
			continue
		}
		if level >= 2 && end.Source["Dial"].IsZero() {
			continue
		}
		if end.connections > 0 {
			end.connections = 0
			end.Disconnected = time.Now()
		}
		e.Ends[ip] = end
	}
	return e
}

// Persist the endpoints to file with the specified parameters. Runs cleanup first.
func (epm *Endpoints) Persist(path string, level uint, min time.Duration, cutoff time.Duration) error {
	if path == "" {
		return nil
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	epm.Cleanup(cutoff)
	e := epm.trim(level, min, cutoff)

	persist, err := json.Marshal(e)
	if err != nil {
		return err
	}

	_, err = writer.Write(persist)
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (e *endpoint) timeConnected() time.Duration {
	if e.Connected.IsZero() {
		return 0
	}
	if e.Disconnected.IsZero() {
		return time.Now().Sub(e.Connected)
	}
	return e.Disconnected.Sub(e.Connected)
}
