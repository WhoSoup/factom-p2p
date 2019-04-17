package p2p

// PeerDistance is a sorting metric for sorting peers by location relative to a specific peer
type PeerDistance struct {
	Pivot  uint32
	Sorted []IP
}

func (pd *PeerDistance) Len() int {
	return len(pd.Sorted)
}

func (pd *PeerDistance) Swap(i, j int) {
	pd.Sorted[i], pd.Sorted[j] = pd.Sorted[j], pd.Sorted[i]
}

func (pd *PeerDistance) Less(i, j int) bool {
	return pd.Distance(i) < pd.Distance(j)
}

func (pd *PeerDistance) Distance(i int) uint32 {
	return uintDistance(pd.Pivot, pd.Sorted[i].Location)
}

// uintDistance returns the absolute difference between i and j
func uintDistance(i, j uint32) uint32 {
	if i > j {
		return i - j
	}
	return j - i
}
