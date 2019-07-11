package p2p

import (
	"time"
)

// Cat = Cyclic Auto Truncate
type cat struct {
	net        *Network
	peerStatus chan peerStatus
	peers      *PeerStore
}

func newCat(net *Network) *cat {
	c := new(cat)
	c.net = net
	c.peerStatus = make(chan peerStatus, 10)
	c.peers = NewPeerStore()
	return c
}

func (c *cat) reseed() {

}

func (c *cat) cycle() {
	for {
		if uint(c.peers.Total()) <= c.net.conf.MinReseed {
			c.reseed()
		}

		time.Sleep(c.net.conf.RoundTime)
	}
}
