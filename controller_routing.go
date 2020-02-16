package p2p

import "time"

// route takes messages from ToNetwork and routes it to the appropriate peers
func (c *controller) route() {
	for {
		// blocking read on ToNetwork, and c.stopRoute
		select {
		case parcel := <-c.net.ToNetwork:
			switch parcel.Address {
			case FullBroadcast:
				for _, p := range c.peers.Slice() {
					p.Send(parcel)
				}

			case Broadcast:
				selection := c.selectBroadcastPeers(c.net.conf.Fanout)
				for _, p := range selection {
					p.Send(parcel)
				}

			case "":
				fallthrough
			case RandomPeer:
				if random := c.randomPeer(); random != nil {
					random.Send(parcel)
				} else {
					c.logger.Debugf("attempted to send parcel %s to a random peer but no peers are connected", parcel)
				}

			default:
				if p := c.peers.Get(parcel.Address); p != nil {
					p.Send(parcel)
				}
			}
		}
	}
}

// manageData processes parcels arriving from peers and responds appropriately.
// application messages are forwarded to the network channel.
func (c *controller) manageData() {
	c.logger.Debug("Start manageData()")
	defer c.logger.Debug("Stop manageData()")
	for {
		select {
		case pp := <-c.peerData:
			parcel := pp.parcel
			peer := pp.peer

			if peer == nil && !parcel.IsApplicationMessage() { // peer disconnected between sending message and now
				c.logger.Debugf("Received parcel %s from peer not in system", parcel)
				continue
			}

			//c.logger.Debugf("Received parcel %s from %s", parcel, peer)
			switch parcel.Type {
			case TypePing:
				go func() {
					parcel := newParcel(TypePong, []byte("Pong"))
					peer.Send(parcel)
				}()
			case TypeMessage:
				//c.net.FromNetwork.Send(parcel)
				fallthrough
			case TypeMessagePart:
				parcel.Type = TypeMessage
				c.net.FromNetwork.Send(parcel)
			case TypePeerRequest:
				if time.Since(peer.lastPeerRequest) >= c.net.conf.PeerRequestInterval {
					peer.lastPeerRequest = time.Now()
					share := c.makePeerShare(peer.Endpoint)
					go c.sharePeers(peer, share)
				} else {
					c.logger.Warnf("peer %s sent a peer request too early", peer)
				}
			case TypePeerResponse:
				c.shareMtx.RLock()
				if async, ok := c.shareListener[peer.Hash]; ok {
					async <- parcel
				}
				c.shareMtx.RUnlock()
			default:
				//not handled
			}
		}
	}
}

func (c *controller) randomPeerConditional(condition func(*Peer) bool) *Peer {
	peers := c.peers.Slice()

	filtered := make([]*Peer, 0)
	for _, p := range peers {
		if condition(p) {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) > 0 {
		return filtered[c.net.rng.Intn(len(filtered))]
	}
	return nil
}

func (c *controller) randomPeer() *Peer {
	peers := c.peers.Slice()
	if len(peers) > 0 {
		return peers[c.net.rng.Intn(len(peers))]
	}
	return nil
}

func (c *controller) selectBroadcastPeers(max uint) []*Peer {
	peers := c.peers.Slice()

	// not enough to randomize
	if uint(len(peers)) <= max {
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

	if uint(len(regular)) < max {
		return append(special, regular...)
	}

	c.net.rng.Shuffle(len(regular), func(i, j int) {
		regular[i], regular[j] = regular[j], regular[i]
	})

	return append(special, regular[:max]...)
}
