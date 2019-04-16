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

// NewPeerList creates a new, empty PeerList
func NewPeerList() *PeerList {
	return new(PeerList)
}

func (pl *PeerList) Find(hash string) *Peer {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	for _, p := range pl.list {
		if p.Hash == hash {
			return p
		}
	}
	return nil
}

// Add adds a given peer to the list
func (pl *PeerList) Add(p *Peer) {
	if p == nil {
		return
	}
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	pl.list = append(pl.list, p)
}

// Remove removes the FIRST instance (by hash) of peer it encounters
func (pl *PeerList) Remove(needle *Peer) {
	if needle == nil {
		return
	}
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	for i, p := range pl.list {
		if p == needle { // pointer comparison
			pl.list[i] = pl.list[len(pl.list)-1] // overwrite deleted peer
			pl.list[len(pl.list)-1] = nil        // free reference
			pl.list = pl.list[:len(pl.list)-1]   // truncate
			return
		}
	}
}

// Slice creates a concurrency safe slice to iterate over
func (pl *PeerList) Slice() []*Peer {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	return append(pl.list[:0:0], pl.list...)
}

func (pl *PeerList) Len() int {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	return len(pl.list)
}

/*
// Search checks if the list contains an entry for the given address and port
//
// returns (peer, true) on success, (nil, false) if nothing was found
func (pl *PeerList) Search(addr, port string) (*Peer, bool) {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	for _, p := range pl.list {
		if p.Address == addr && p.Port == port {
			return p, true
		}
	}
	return nil, false
}

func (pl *PeerList) SearchUnused(addr string) *Peer {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	for _, p := range pl.list {
		if p.Address == addr && p.NodeID == 0 {
			return p
		}
	}
	return nil
}

func (pl *PeerList) IsConnected(addr string) bool {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()
	for _, p := range pl.list {
		if p.Address == addr {
			return true
		}
	}
	return false
}
*/
