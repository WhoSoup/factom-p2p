package p2p

import (
	"sync"
	"time"
)

type CatStore struct {
	peers    map[string]*Peer // hash -> peer
	counts   map[string]int   // address[:port] -> count
	curSlice []*Peer          // temporary slice that gets reset when changes are made

	incoming int
	outgoing int

	Bans map[string]time.Time `json:"bans"`

	special map[string]bool // (address|address:port) -> bool

	mtx sync.RWMutex
}

type catpoint struct {
	IP           IP                   `json:"ip"`
	Seen         time.Time            `json:"seen"`
	Source       map[string]time.Time `json:"source"`
	Connected    time.Time            `json:"connected"`
	Disconnected time.Time            `json:"disconnected"`
	connections  uint
	lock         time.Time
}

func (cs *CatStore) Add(peer *Peer) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	cs.peers[peer.Hash] = peer
	cs.counts[peer.IP.Address]++
	cs.counts[peer.IP.String()]++

	if peer.IsIncoming {
		cs.incoming++
	} else {
		cs.outgoing++
	}

	cs.curSlice = nil
}

func (cs *CatStore) Remove(peer *Peer) {

}
