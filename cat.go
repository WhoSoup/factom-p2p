package p2p

import (
	"time"
)

// Cat = Cyclic Auto Truncate
type cat struct {
	net        *Network
	peerStatus chan peerStatus
	peers      *PeerStore
	queue      chan IP
}

func newCat(net *Network) *cat {
	c := new(cat)
	c.net = net
	c.peerStatus = make(chan peerStatus, 10)
	c.queue = make(chan IP, 50)
	c.peers = NewPeerStore()
	return c
}

func (c *cat) queueLoop() {

}

func (c *cat) reseed() {
	ips := c.net.seed.retrieve()
	for _, ip := range ips {
		c.queue <- ip
	}
}

func (c *cat) drop(amount uint) {

}

func (c *cat) cycle() {
	for {
		t := uint(c.peers.Total())
		if t <= c.net.conf.MinReseed {
			c.reseed()
		}

		if t >= c.net.conf.Target {
			c.drop(t - c.net.conf.Target + c.net.conf.Drop)
		}

		time.Sleep(c.net.conf.RoundTime)
	}
}
