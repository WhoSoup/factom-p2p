package p2p

// PeerDistance is a sorting metric for sorting peers by location relative to a specific peer
type PeerDistance struct {
	origin *Peer
	sorted []*Peer
}

func (pd *PeerDistance) Len() int {
	return len(pd.sorted)
}

func (pd *PeerDistance) Swap(i, j int) {
	pd.sorted[i], pd.sorted[j] = pd.sorted[j], pd.sorted[i]
}

func (pd *PeerDistance) Less(i, j int) bool {
	return uintDistance(pd.origin.Location, pd.sorted[i].Location) < uintDistance(pd.origin.Location, pd.sorted[j].Location)
}

// uintDistance returns the absolute difference between i and j
func uintDistance(i, j uint32) uint32 {
	if i > j {
		return i - j
	}
	return j - i
}
