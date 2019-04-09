package p2p

import "sync"

// PeerList is a concurrency safe way to manage a list of peers
//
// The order of the peers inside the list is NOT guaranteed and will change
// when peers are removed
type PeerList struct {
	list  []*Peer
	mutex sync.RWMutex
}

func NewPeerList() PeerList {
	return PeerList{}
}

func (pl PeerList) Add(p *Peer) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	pl.list = append(pl.list, p)
}

func (pl PeerList) Remove(needle *Peer) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	for i, p := range pl.list {
		if p.Hash == needle.Hash {
			pl.list[i] = pl.list[len(pl.list)-1] // overwrite deleted peer
			pl.list[len(pl.list)-1] = nil        // free reference
			pl.list = pl.list[:len(pl.list)-1]   // truncate
			return
		}
	}
}

// Slice creates a concurrency safe slice to iterate over
func (pl PeerList) Slice() []*Peer {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	return append(pl.list[:0:0], pl.list...)
}

func (pl PeerList) HasIPPort(addr, port string) (*Peer, bool) {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	for _, p := range pl.list {
		if p.Address == addr && p.Port == port {
			return p, true
		}
	}
	return nil, false
}
