package p2p

import "sync"

// [address][nodeid] = peer
type PeerStore struct {
	mtx       sync.RWMutex
	peers     map[string]map[uint64]*Peer
	connected map[string]int // address -> count
	curSlice  []*Peer
	Incoming  int
	Outgoing  int
}

func NewPeerStore() *PeerStore {
	ps := new(PeerStore)
	ps.peers = make(map[string]map[uint64]*Peer)
	ps.connected = make(map[string]int)
	return ps
}

func (ps *PeerStore) makeAddr(addr string) {
	if _, ok := ps.peers[addr]; !ok {
		ps.peers[addr] = make(map[uint64]*Peer)
	}
}

func (ps *PeerStore) Replace(p *Peer) *Peer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	ps.makeAddr(p.Address)
	old := ps.peers[p.Address][p.NodeID]
	ps.curSlice = nil
	ps.peers[p.Address][p.NodeID] = p
	ps.connected[p.Address]++

	if p.IsIncoming {
		ps.Incoming++
	} else {
		ps.Outgoing++
	}

	if old != nil {
		ps.connected[p.Address]--
		if old.IsIncoming {
			ps.Incoming--
		} else {
			ps.Outgoing--
		}
	}
	return old
}

func (ps *PeerStore) Remove(p *Peer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	if _, ok := ps.peers[p.Address]; ok {
		old, ok := ps.peers[p.Address][p.NodeID]
		if ok {
			ps.connected[p.Address]--
			if old.IsIncoming {
				ps.Incoming--
			} else {
				ps.Outgoing--
			}
		}
		ps.curSlice = nil
		delete(ps.peers[p.Address], p.NodeID)
	}
}

func (ps *PeerStore) Count() int {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.Incoming + ps.Outgoing
}

func (ps *PeerStore) IsConnected(addr string) bool {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()
	return ps.connected[addr] > 0
}

func (ps *PeerStore) Slice() []*Peer {
	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	if ps.curSlice != nil {
		return ps.curSlice
	}
	r := make([]*Peer, 0)
	for _, addr := range ps.peers {
		for _, p := range addr {
			r = append(r, p)
		}
	}
	ps.curSlice = r
	return r
}
