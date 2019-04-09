package p2p

import "sync"

// PeerMap holds peers in a set of different structures to allow for efficient
// retrieval in all circumstances
type PeerMap struct {
	lock    sync.RWMutex
	bySlice PeerList
	byHash  map[string]*Peer
	byIP    map[string]PeerList
}

func NewPeerMap() *PeerMap {
	n := new(PeerMap)
	n.byHash = make(map[string]*Peer)
	n.byIP = make(map[string]PeerList)
	n.bySlice = NewPeerList()
	return n
}

func (pm *PeerMap) Add(p *Peer) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.byHash[p.Hash] = p

	if _, ok := pm.byIP[p.Address]; !ok {
		pm.byIP[p.Address] = NewPeerList()
	}
	pm.byIP[p.Address].Add(p)
	pm.bySlice.Add(p)
}

func (pm *PeerMap) Remove(p *Peer) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	delete(pm.byHash, p.Hash)
	pm.byIP[p.Address].Remove(p)
	pm.bySlice.Remove(p)
}

func (pm *PeerMap) Has(hash string) bool {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	return pm.byHash[hash] != nil
}

func (pm *PeerMap) Get(hash string) *Peer {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	return pm.byHash[hash]
}

func (pm *PeerMap) Slice() []*Peer {
	return pm.bySlice.Slice()
}

func (pm *PeerMap) HasIPPort(addr, port string) (*Peer, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	if list, ok := pm.byIP[addr]; ok {
		return list.HasIPPort(addr, port)
	}
	return nil, false
}

func (pm *PeerMap) IsConnected(address string) bool {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	if list, ok := pm.byIP[address]; ok {
		for _, p := range list.Slice() {
			if !p.IsOffline() {
				return true
			}
		}
	}
	return false

}
