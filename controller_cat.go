package p2p

import (
	"fmt"
	"time"
)

// processPeers processes a peer share response
func (c *controller) processPeerShare(peer *Peer, parcel *Parcel) []Endpoint {
	list, err := peer.prot.ParsePeerShare(parcel.Payload)

	if err != nil {
		c.logger.WithError(err).Warnf("Failed to unmarshal peer share from peer %s", peer)
	}

	c.logger.Debugf("Received peer share from %s: %+v", peer, list)

	// cycles through list twice but we don't want to add any if one of them is bad
	for _, p := range list {
		if !p.Verify() {
			c.logger.Infof("Peer %s tried to send us peer share with bad data: %s", peer, p)
			return nil
		}
	}

	var res []Endpoint
	for _, p := range list {
		ep, err := NewEndpoint(p.IP, p.Port)
		if err != nil {
			c.logger.WithError(err).Infof("Unable to register endpoint %s:%s from peer %s", p.IP, p.Port, peer)
		} else if !c.isBannedEndpoint(ep) {
			//c.endpoints.Register(ip, peer.IP.Address)
			res = append(res, ep)
		}
	}

	if c.net.prom != nil {
		c.net.prom.KnownPeers.Set(float64(c.peers.Total()))
	}

	return res
}

func (c *controller) trimShare(list []Endpoint, shuffle bool) []Endpoint {
	if len(list) == 0 {
		return nil
	}
	if shuffle {
		c.net.rng.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
	}
	if uint(len(list)) > c.net.conf.PeerShareAmount {
		list = list[:c.net.conf.PeerShareAmount]
	}
	return list
}

func (c *controller) makePeerShare(ep Endpoint) []Endpoint {
	var list []Endpoint
	tmp := c.peers.Slice()
	for _, i := range c.net.rng.Perm(len(tmp)) {
		if tmp[i].Endpoint == ep {
			continue
		}
		list = append(list, tmp[i].Endpoint)
		if uint(len(tmp)) >= c.net.conf.PeerShareAmount {
			break
		}
	}
	return list
}

// sharePeers creates a list of peers to share and sends it to peer
func (c *controller) sharePeers(peer *Peer) {
	if peer == nil {
		return
	}
	// CAT select n random active peers
	list := c.makePeerShare(peer.Endpoint)
	payload, err := peer.prot.MakePeerShare(list)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal peer list to json")
		return
	}
	c.logger.Debugf("Sharing %d peers with %s", len(list), peer)
	parcel := newParcel(TypePeerResponse, payload)
	peer.Send(parcel)
}

func (c *controller) catRound() {
	c.logger.Debug("Cat Round")
	c.rounds++
	peers := c.peers.Slice()
	toDrop := len(peers) - int(c.net.conf.Drop)

	if toDrop > 0 {
		perm := c.net.rng.Perm(len(peers))

		dropped := 0
		for _, i := range perm {
			if c.isSpecial(peers[i].Endpoint) {
				continue
			}
			peers[i].Stop()
			dropped++
			if dropped >= toDrop {
				break
			}
		}
	}
}

// this function is only intended to be run single-threaded inside the replenish loop
// it works by creating a closure that contains a channel specific for this call
// the closure is called in controller.manageData
// if there is no response from the peer after 5 seconds, it times out
func (c *controller) asyncPeerRequest(peer *Peer) ([]Endpoint, error) {
	c.shareMtx.Lock()

	var share []Endpoint
	async := make(chan bool, 1)
	f := func(parcel *Parcel) {
		share = c.trimShare(c.processPeerShare(peer, parcel), true)
		async <- true
	}
	c.shareListener[peer] = f
	c.shareMtx.Unlock()

	defer func() {
		c.shareMtx.Lock()
		if ff, ok := c.shareListener[peer]; ok && &ff == &f {
			delete(c.shareListener, peer)
		} else {
			fmt.Printf("DEBUG comparing f&ff: %v == %v\n", &f, &ff)
		}
		c.shareMtx.Unlock()
	}()

	req := newParcel(TypePeerRequest, []byte("Peer Request"))
	peer.Send(req)

	select {
	case <-async:
	case <-time.After(time.Second * 5):
		return nil, fmt.Errorf("timeout")
	}

	return share, nil
}

// catReplenish is the loop that brings the node up to the desired number of connections.
// Does nothing if we have enough peers, otherwise it sends a peer request to a random peer.
func (c *controller) catReplenish() {
	c.logger.Debug("Replenish loop started")
	defer c.logger.Debug("Replenish loop ended")
	for {
		if uint(c.peers.Total()) >= c.net.conf.Target {
			time.Sleep(time.Second)
			continue
		}

		var connect []Endpoint

		if uint(c.peers.Total()) <= c.net.conf.MinReseed {
			seeds := c.seed.retrieve()
			for _, s := range seeds {
				if c.peers.Connected(s) {
					continue
				}
				connect = append(connect, s)
			}
		}

		if len(connect) == 0 {
			if p := c.randomPeer(); p != nil {
				// error just means timeout of async request
				if eps, err := c.asyncPeerRequest(p); err == nil {
					for _, ep := range eps {
						if c.peers.Connected(ep) {
							continue
						}
						connect = append(connect, ep)
					}
				}
			}
		}

		for _, ep := range connect {
			c.Dial(ep)
		}
	}
}

func (c *controller) selectBroadcastPeers(count uint) []*Peer {
	peers := c.peers.Slice()

	// not enough to randomize
	if uint(len(peers)) <= count {
		return peers
	}

	var special []*Peer
	var regular []*Peer

	for _, p := range peers {
		if c.isSpecial(p.Endpoint) {
			special = append(special, p)
		} else {
			regular = append(regular, p)
		}
	}

	if uint(len(regular)) < count {
		return append(special, regular...)
	}

	c.net.rng.Shuffle(len(regular), func(i, j int) {
		regular[i], regular[j] = regular[j], regular[i]
	})

	return append(special, peers[:count]...)
}
