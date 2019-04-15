package p2p

type PartialPeerView struct {
	view   map[string]bool
	source map[string]map[string]bool
}

func NewPartialPeerView() *PartialPeerView {
	ppv := new(PartialPeerView)
	ppv.view = make(map[string]bool)
	ppv.source = make(map[string]map[string]bool)
	return ppv
}
