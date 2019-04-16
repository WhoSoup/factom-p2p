package p2p

import "sync"

// PeerMap holds peers in a set of different structures to allow for efficient
// retrieval in all circumstances
//
// This map does not allow duplicate peers. A peer is considered equal if they have the same peer hash
type PeerMap struct {
	lock    sync.RWMutex
	bySlice *PeerList
	byHash  map[string]*Peer
	byIP    map[string]*PeerList
}

// NewPeerMap creates and initializes a new PeerMap
func NewPeerMap() *PeerMap {
	n := new(PeerMap)
	n.byHash = make(map[string]*Peer)
	n.byIP = make(map[string]*PeerList)
	n.bySlice = NewPeerList()
	return n
}

func (pm *PeerMap) Add(p *Peer) {
	if p == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()

	if pm.byHash[p.Hash] != nil {
		return
	}

	pm.byHash[p.Hash] = p

	if _, ok := pm.byIP[p.Address]; !ok {
		pm.byIP[p.Address] = NewPeerList()
	}
	pm.byIP[p.Address].Add(p)
	pm.bySlice.Add(p)
}

// Remove removes the specified peer
func (pm *PeerMap) Remove(p *Peer) {
	if p == nil {
		return
	}
	pm.lock.Lock()
	defer pm.lock.Unlock()
	if pm.byHash[p.Hash] == nil {
		return
	}
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

// Search finds the first peer with the specified address and port
func (pm *PeerMap) Search(addr, port string) (*Peer, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	if list, ok := pm.byIP[addr]; ok {
		return list.Search(addr, port)
	}
	return nil, false
}

func (pm *PeerMap) SearchUnused(addr string) *Peer {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	if list, ok := pm.byIP[addr]; ok {
		return list.SearchUnused(addr)
	}
	return nil
}

func (pm *PeerMap) SearchDuplicate(needle *Peer) *Peer {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	if list, ok := pm.byIP[needle.Address]; ok {
		for _, p := range list.Slice() {
			if p.Hash != needle.Hash && p.Port == needle.Port { // address already matches
				return p
			}
		}
	}
	return nil
}

func (pm *PeerMap) SearchDuplicateNodeID(needle *Peer) *Peer {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	if list, ok := pm.byIP[needle.Address]; ok {
		for _, p := range list.Slice() {
			if p.Hash != needle.Hash && p.NodeID == needle.NodeID { // address already matches
				return p
			}
		}
	}
	return nil
}

// IsConnected returns true if at least one peer with the given address is online or connecting
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

// ConnectedCount returns the number of online or connecting peers from the given address
func (pm *PeerMap) ConnectedCount(address string) int {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	count := 0
	if list, ok := pm.byIP[address]; ok {
		for _, p := range list.Slice() {
			if !p.IsOffline() {
				count++
			}
		}
	}
	return count
}
