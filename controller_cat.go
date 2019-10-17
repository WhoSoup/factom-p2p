package p2p

import "time"

// processPeers processes a peer share response
func (c *controller) processPeers(peer *Peer, parcel *Parcel) []Endpoint {
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

// sharePeers creates a list of peers to share and sends it to peer
func (c *controller) sharePeers(peer *Peer) {
	if peer == nil {
		return
	}
	// CAT select n random active peers
	var list []Endpoint
	tmp := c.peers.Slice()
	for _, i := range c.net.rng.Perm(len(tmp)) {
		if tmp[i].Hash == peer.Hash {
			continue
		}
		list = append(list, tmp[i].Endpoint)
		if uint(len(tmp)) >= c.net.conf.PeerShareAmount {
			break
		}
	}

	payload, err := peer.prot.MakePeerShare(list)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal peer list to json")
		return
	}
	c.logger.Debugf("Sharing %d peers with %s", len(list), peer)
	parcel := newParcel(TypePeerResponse, payload)
	peer.Send(parcel)
}

func (c *controller) reseed() {
	if uint(c.peers.Total()) < c.net.conf.MinReseed {
		seeds := c.seed.retrieve()
		for _, endpoint := range seeds {
			select {
			case c.dial <- endpoint:
			default:
			}
		}
	}
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

// catReplenish is the loop that brings the node up to the desired number of connections.
// Does nothing if we have enough peers, otherwise it sends a peer request to a random peer.
func (c *controller) catReplenish() {
	c.logger.Debug("Replenish loop started")
	defer c.logger.Debug("Replenish loop ended")
	for {

		if uint(c.peers.Total()) >= c.net.conf.Target {
			time.Sleep(time.Second * 5)
			continue
		}

		if uint(c.peers.Total()) <= c.net.conf.MinReseed {
			c.reseed()
			time.Sleep(time.Second * 2)
		}

		p := c.randomPeer()

		if p == nil { // no peers connected
			time.Sleep(time.Second)
			continue
		}

		async := make(chan *Parcel, 1)
		p.peerShareDeliver = async

		req := newParcel(TypePeerRequest, []byte("Peer Request"))
		p.Send(req)

		select {
		case resp := <-async:
			eps := c.trimShare(c.processPeers(p, resp), true)
			for _, ep := range eps {
				select {
				case c.dial <- ep:
				default:
				}
			}
		case <-time.After(time.Second * 5):
		}
		p.peerShareDeliver = nil
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
