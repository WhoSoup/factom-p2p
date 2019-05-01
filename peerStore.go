package p2p

import (
	"fmt"
	"sync"
)

// PeerStore holds active Peers, managing them in a concurrency safe
// manner and providing lookup via various functions
type PeerStore struct {
	mtx       sync.RWMutex
	peers     map[string]*Peer // hash -> peer
	connected map[string]int   // address -> count
	curSlice  []*Peer
	Incoming  int
	Outgoing  int
}

// NewPeerStore initializes a new peer store
func NewPeerStore() *PeerStore {
	ps := new(PeerStore)
	ps.peers = make(map[string]*Peer)
	ps.connected = make(map[string]int)
	return ps
}

/*
// Replace adds the given peer to the list of managed peers.
// If a peer with that hash already existed, it returns the old one
func (ps *PeerStore) Replace(p *Peer) *Peer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	old := ps.peers[p.Hash]
	ps.curSlice = nil
	ps.peers[p.Hash] = p
	ps.connected[p.IP.Address]++

	if p.IsIncoming {
		ps.Incoming++
	} else {
		ps.Outgoing++
	}

	if old != nil {
		ps.connected[p.IP.Address]--
		if old.IsIncoming {
			ps.Incoming--
		} else {
			ps.Outgoing--
		}
	}
	return old
}*/

func (ps *PeerStore) Add(p *Peer) error {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	_, ok := ps.peers[p.Hash]
	if ok {
		return fmt.Errorf("peer already exists")
	}
	ps.curSlice = nil
	ps.peers[p.Hash] = p
	ps.connected[p.IP.Address]++

	if p.IsIncoming {
		ps.Incoming++
	} else {
		ps.Outgoing++
	}
	return nil
}

func (ps *PeerStore) Remove(p *Peer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	if old, ok := ps.peers[p.Hash]; ok && old == p { // pointer comparison
		ps.connected[p.IP.Address]--
		if old.IsIncoming {
			ps.Incoming--
		} else {
			ps.Outgoing--
		}
		ps.curSlice = nil
		delete(ps.peers, p.Hash)
	}
}

func (ps *PeerStore) Total() int {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return len(ps.peers)
}

func (ps *PeerStore) Unique() int {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return len(ps.connected)
}

func (ps *PeerStore) Get(hash string) *Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.peers[hash]
}

func (ps *PeerStore) IsConnected(addr string) bool {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.connected[addr] > 0
}

func (ps *PeerStore) Count(addr string) int {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.connected[addr]
}

func (ps *PeerStore) Slice() []*Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	if ps.curSlice != nil {
		return ps.curSlice
	}
	r := make([]*Peer, 0)
	for _, p := range ps.peers {
		r = append(r, p)
	}
	ps.curSlice = r
	return r
}
