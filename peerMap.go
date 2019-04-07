package p2p

import "sync"

type PeerMap struct {
	lock    sync.RWMutex
	bySlice []*Peer
	byHash  map[string]*Peer
	byIP    map[string]map[string]*Peer // TODO make this a list
}

func NewPeerMap() *PeerMap {
	n := new(PeerMap)
	n.byHash = make(map[string]*Peer)
	n.byIP = make(map[string]map[string]*Peer)
	return n
}

func (pm *PeerMap) Add(p *Peer) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.byHash[p.Hash] = p

	if _, ok := pm.byIP[p.Address]; !ok {
		pm.byIP[p.Address] = make(map[string]*Peer)
	}
	pm.byIP[p.Address][p.Port] = p
	pm.bySlice = append(pm.bySlice, p)
}

func (pm *PeerMap) Remove(p *Peer) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	delete(pm.byHash, p.Hash)
	delete(pm.byIP, p.Address)
	for i, x := range pm.bySlice {
		if x.Hash == p.Hash {
			pm.bySlice[i] = pm.bySlice[len(pm.bySlice)-1]
			pm.bySlice = pm.bySlice[:len(pm.bySlice)-1]
			break
		}
	}
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
	return pm.bySlice[:]
}
