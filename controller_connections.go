package p2p

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// manageOnline listens to peerStatus updates sent out by peers
// if a peer notifies it's going offline, it will be removed
// if a peer notifies it's coming online, existing peers with the same hash are removed
func (c *controller) manageOnline() {
	c.logger.Debug("Start manageOnline()")
	defer c.logger.Debug("Stop manageOnline()")
	for {
		select {
		case pc := <-c.peerStatus:
			if pc.online {
				if old := c.peers.Get(pc.peer.Hash); old != nil {
					old.Stop()
					c.logger.Debugf("removing old peer %s", pc.peer.Hash)
					c.peers.Remove(old)
				}
				err := c.peers.Add(pc.peer)
				if err != nil {
					c.logger.WithError(err).Errorf("Unable to add peer %s", pc.peer)
				}
				c.logger.Debugf("adding peer %s", pc.peer.Hash)
			} else {
				c.peers.Remove(pc.peer)
				c.logger.Debugf("removing peer %s", pc.peer.Hash)
			}
			if c.net.prom != nil {
				c.net.prom.Connections.Set(float64(c.peers.Total()))
				//c.net.prom.Unique.Set(float64(c.peers.Unique()))
				c.net.prom.Incoming.Set(float64(c.peers.Incoming()))
				c.net.prom.Outgoing.Set(float64(c.peers.Outgoing()))
			}
		}
	}
}

// preliminary check to see if we should accept an unknown connection
func (c *controller) allowIncoming(addr string) error {
	if c.isBannedIP(addr) {
		return fmt.Errorf("Address %s is banned", addr)
	}

	if uint(c.peers.Total()) >= c.net.conf.MaxIncoming && !c.isSpecialIP(addr) {
		return fmt.Errorf("Refusing incoming connection from %s because we are maxed out (%d of %d)", addr, c.peers.Total(), c.net.conf.MaxIncoming)
	}

	if c.net.conf.PeerIPLimitIncoming > 0 && uint(c.peers.Count(addr)) >= c.net.conf.PeerIPLimitIncoming {
		return fmt.Errorf("Rejecting %s due to per ip limit of %d", addr, c.net.conf.PeerIPLimitIncoming)
	}

	return nil
}

// handshakeIncoming performs the handshake maneouver for incoming connections.
// 	1. Determine their protocol from the first message they send
//	2. If we understand that protocol, validate that handshake
//	3. Reply with a handshake
//	4. Create a peer with that protocol
//
// If the incoming endpoint is banned, the connection is closed without alternatives.
// If the node is full, the connection is closed with alternatives.
//
// For more information, see the README
func (c *controller) handshakeIncoming(con net.Conn) {
	if c.net.prom != nil {
		c.net.prom.Connecting.Inc()
		defer c.net.prom.Connecting.Dec()
	}

	host, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		c.logger.WithError(err).Debugf("Unable to parse address %s", con.RemoteAddr().String())
		con.Close()
		return
	}

	// port is overriden during handshake, use default port as temp port
	ep, err := NewEndpoint(host, c.net.conf.ListenPort)
	if err != nil { // should never happen for incoming
		c.logger.WithError(err).Debugf("Unable to decode address %s", host)
		con.Close()
		return
	}

	timeout := time.Now().Add(c.net.conf.HandshakeTimeout)
	con.SetDeadline(timeout)

	// reject incoming connections based on host
	if err = c.allowIncoming(host); err != nil {
		c.logger.WithError(err).Infof("Rejecting connection")
		share := c.makePeerShare(ep)  // they're not connected to us, so we don't have them in our system
		c.RejectWithShare(con, share) // closes con
		return
	}

	// upgrade connection to a metrics connection
	metrics := NewMetricsReadWriter(con)
	prot, handshake, err := c.detectProtocolFromFirstMessage(metrics)
	if err != nil {
		c.logger.WithError(err).Debug("error detecting protocol")
		con.Close()
		return
	}

	if err := handshake.Valid(c.net.conf, c.net.instanceID); err != nil {
		c.logger.WithError(err).Debugf("inbound connection from %s failed handshake", host)
		con.Close()
		return
	}

	reply := newHandshake(c.net.conf, c.net.instanceID)
	reply.Version = handshake.Version
	if err := prot.SendHandshake(reply); err != nil {
		c.logger.WithError(err).Debugf("unable to reply to handshake")
		con.Close()
		return
	}

	// listenport has been validated in handshake.Valid
	ep.Port = handshake.ListenPort

	peer := newPeer(c.net, handshake.NodeID, ep, con, prot, metrics, true)
	c.peerStatus <- peerStatus{peer: peer, online: true}

	// a p2p1 node sends a peer request, so it needs to be processed
	if handshake.Type == TypePeerRequest {
		req := newParcel(TypePeerRequest, []byte("Peer Request"))
		req.Address = peer.Hash
		c.peerData <- peerParcel{peer: peer, parcel: req}
	}

	c.logger.Debugf("Incoming handshake success for peer %s, version %s", peer.Hash, peer.prot)
}

// detectProtocol will listen for data to arrive on the ReadWriter and then attempt to interpret it.
// the existing protocol is only needed for nodes running v9 in order to bring
func (c *controller) detectProtocolFromFirstMessage(rw io.ReadWriter) (Protocol, *Handshake, error) {
	var prot Protocol
	var handshake *Handshake

	buffy := bufio.NewReader(rw)

	sig, err := buffy.Peek(4)
	if err != nil {
		return nil, nil, err
	}

	if bytes.Equal(sig, V11Signature) {
		prot = newProtocolV11(bufio.NewReadWriter(buffy, bufio.NewWriter(rw)))
		hs, err := prot.ReadHandshake()
		if err != nil {
			return nil, nil, err
		}

		if err := hs.Valid(c.net.conf, c.net.instanceID); err != nil {
			return nil, nil, err
		}
		handshake = hs
	} else {
		decoder := gob.NewDecoder(buffy)
		encoder := gob.NewEncoder(rw)

		v9test := newProtocolV9(c.net.conf.Network, c.net.conf.NodeID, c.net.conf.ListenPort, decoder, encoder)
		hs, err := v9test.ReadHandshake()
		if err != nil {
			return nil, nil, err
		}

		if err := hs.Valid(c.net.conf, c.net.instanceID); err != nil {
			return nil, nil, err
		}

		v := hs.Version
		if v > c.net.conf.ProtocolVersion {
			v = c.net.conf.ProtocolVersion
		}

		handshake = hs

		switch v {
		case 9:
			prot = v9test
		case 10:
			prot = newProtocolV10(decoder, encoder)
		default:
			return nil, nil, fmt.Errorf("unsupported protocol version %d", v)
		}
	}

	return prot, handshake, nil
}

func (c *controller) selectProtocol(rw io.ReadWriter) Protocol {
	switch c.net.conf.ProtocolVersion {
	case 11:
		return newProtocolV11(rw)
	case 10:
		decoder := gob.NewDecoder(rw)
		encoder := gob.NewEncoder(rw)
		return newProtocolV10(decoder, encoder)
	default:
		decoder := gob.NewDecoder(rw)
		encoder := gob.NewEncoder(rw)
		return newProtocolV9(c.net.conf.Network, c.net.conf.NodeID, c.net.conf.ListenPort, decoder, encoder)
	}
}

// handshakeOutgoing performs the handshake maneouver when dialing to remote nodes.
// 	1. Pick our desired protocol
//	2. Send a handshake
//	3. Figure out which protocol to use from the reply
//	4. Create a peer if a compatible protocol is established
//
// It is possible the endpoint will reject due to being full, in which
// case this function returns an error AND a list of alternate endpoints
//
// For more information, see the README
func (c *controller) handshakeOutgoing(ep Endpoint, con net.Conn) (*Peer, []Endpoint, error) {
	tmplogger := c.logger.WithField("endpoint", ep)
	timeout := time.Now().Add(c.net.conf.HandshakeTimeout)
	con.SetDeadline(timeout)

	handshake := newHandshake(c.net.conf, c.net.instanceID)
	metrics := NewMetricsReadWriter(con)
	desiredProt := c.selectProtocol(metrics)

	failfunc := func(err error) (*Peer, []Endpoint, error) {
		tmplogger.WithError(err).Debug("Handshake failed")
		con.Close()
		return nil, nil, err
	}

	if err := desiredProt.SendHandshake(handshake); err != nil {
		return failfunc(err)
	}

	prot, reply, err := c.detectProtocolFromFirstMessage(metrics)
	if err != nil {
		return failfunc(err)
	}

	// this is required because a new protocol is instantiated in the above call
	// since V9Msg is already registered in the other end's gob, a new gob encoder
	// would try to register it again, causing an error on the other side
	// v10 is fine since it switches to a new V10Msg
	if v9, ok := desiredProt.(*ProtocolV9); ok {
		if v9new, ok := prot.(*ProtocolV9); ok {
			v9new.encoder = v9.encoder
		}
	}

	// dialed a node that's full
	if reply.Type == TypeRejectAlternative {
		con.Close()
		tmplogger.Debug("con rejected with alternatives")
		return nil, reply.Alternatives, fmt.Errorf("connection rejected")
	}

	peer := newPeer(c.net, reply.NodeID, ep, con, prot, metrics, false)
	c.peerStatus <- peerStatus{peer: peer, online: true}

	// a p2p1 node sends a peer request, so it needs to be processed
	if reply.Type == TypePeerRequest {
		req := newParcel(TypePeerRequest, []byte("Peer Request"))
		req.Address = peer.Hash
		c.peerData <- peerParcel{peer: peer, parcel: req}
	}

	c.logger.Debugf("Outgoing handshake success for peer %s, version %s", peer.Hash, peer.prot)

	return peer, nil, nil
}

// RejectWithShare rejects an incoming connection by sending them a handshake that provides
// them with alternative peers to connect to
func (c *controller) RejectWithShare(con net.Conn, share []Endpoint) error {
	defer con.Close() // we're rejecting, so always close

	prot := c.selectProtocol(con)

	handshake := newHandshake(c.net.conf, 0)
	handshake.Type = TypeRejectAlternative
	handshake.Alternatives = share

	return prot.SendHandshake(handshake)
}

// Dial attempts to connect to a remote endpoint.
// If the dial was not successful, it may return a list of alternate endpoints
// given by the remote host.
func (c *controller) Dial(ep Endpoint) (bool, []Endpoint) {
	if c.net.prom != nil {
		c.net.prom.Connecting.Inc()
		defer c.net.prom.Connecting.Dec()
	}

	c.logger.Debugf("Dialing to %s", ep)
	con, err := c.dialer.Dial(ep)
	if err != nil {
		c.logger.WithError(err).Infof("Failed to dial to %s", ep)
		return false, nil
	}

	peer, alternatives, err := c.handshakeOutgoing(ep, con)
	if err != nil { // handshake closes connection
		if err.Error() == "loopback" {
			c.logger.Debugf("Banning ourselves for 50 years")
			c.banEndpoint(ep, time.Hour*24*365*50) // ban for 50 years
			return false, nil
		}

		if len(alternatives) > 0 {
			c.logger.Debugf("Connection declined with alternatives from %s", ep)
			return false, alternatives
		}
		c.logger.WithError(err).Debugf("Handshake fail with %s", ep)
		return false, nil
	}

	c.logger.Debugf("Handshake success for peer %s, version %s", peer.Hash, peer.prot)
	return true, nil
}

// listen listens for incoming TCP connections and passes them off to handshake maneuver
func (c *controller) listen() {
	tmpLogger := c.logger.WithFields(log.Fields{"host": c.net.conf.BindIP, "port": c.net.conf.ListenPort})
	tmpLogger.Debug("controller.listen() starting up")

	addr := fmt.Sprintf("%s:%s", c.net.conf.BindIP, c.net.conf.ListenPort)

	l, err := NewLimitedListener(addr, c.net.conf.ListenLimit)
	if err != nil {
		tmpLogger.WithError(err).Error("controller.Start() unable to start limited listener")
		return
	}

	c.listener = l

	// start permanent loop
	// terminates on program exit or when listener is closed
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && !ne.Timeout() {
				if !ne.Temporary() {
					tmpLogger.WithError(err).Warn("controller.acceptLoop() error accepting")
				}
			}
			continue
		}

		go c.handshakeIncoming(conn)
	}
}
