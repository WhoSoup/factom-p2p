package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
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
				old := c.peers.Get(pc.peer.Hash)
				if old != nil {
					old.Stop()
					c.logger.Debugf("removing old peer %s", pc.peer.Hash)
					c.peers.Remove(old)
				}
				err := c.peers.Add(pc.peer)
				if err != nil {
					c.logger.Errorf("Unable to add peer %s to peer store because an old peer still exists", pc.peer)
				}
			} else {
				c.peers.Remove(pc.peer)
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

	if uint(c.peers.Total()) >= c.net.conf.Incoming && !c.isSpecialIP(addr) {
		return fmt.Errorf("Refusing incoming connection from %s because we are maxed out (%d of %d)", addr, c.peers.Total(), c.net.conf.Incoming)
	}

	if c.net.conf.PeerIPLimitIncoming > 0 && uint(c.peers.Count(addr)) >= c.net.conf.PeerIPLimitIncoming {
		return fmt.Errorf("Rejecting %s due to per ip limit of %d", addr, c.net.conf.PeerIPLimitIncoming)
	}

	return nil
}

// what to do with a new tcp connection
func (c *controller) handleIncoming(con net.Conn) {
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

	// if we're full, give them alternatives
	if err = c.allowIncoming(host); err != nil {
		c.logger.WithError(err).Infof("Rejecting connection")
		share := c.makePeerShare(ep)  // they're not connected to us, so we don't have them in our system
		c.RejectWithShare(con, share) // closes con
		return
	}

	// don't bother with alternatives for incoming connections
	peer, _, err := c.handshake(host, con, true)

	if err != nil {
		c.logger.WithError(err).Debugf("inbound connection from %s failed handshake", host)
		con.Close()
		return
	}

	c.logger.Debugf("Incoming handshake success for peer %s, version %s", peer.Hash, peer.prot.Version())
}

// handshake performs a basic handshake maneouver to establish the validity of the connection.
// The functionality is symmetrical and used for both incoming and outgoing connections.
// The handshake is backwards compatible with V9. Since V9 doesn't have an explicit handshake, V10
// sends a V9 parcel upon connection and waits for the response, which can be any parcel.
//
// The handshake ensures that ALL peers have a valid Port field to start with.
// If there is no reply within the specified HandshakeTimeout config setting, the process
// fails
//
// For outgoing connections, it is possible the endpoint will reject due to being full, in which
// case this function returns an error AND a list of alternate endpoints
func (c *controller) handshake(ip string, con net.Conn, incoming bool) (*Peer, []Endpoint, error) {
	tmplogger := c.logger.WithField("addr", ip).WithField("incoming", incoming)
	timeout := time.Now().Add(c.net.conf.HandshakeTimeout)

	nonce := make([]byte, 8) // loopback detection
	binary.LittleEndian.PutUint64(nonce, c.net.instanceID)

	// upgrade connection to a metrics connection
	metrics := NewMetricsReadWriter(con)

	handshake := newHandshake(c.net.conf, nonce)
	decoder := gob.NewDecoder(metrics) // pipe gob through the metrics writer
	encoder := gob.NewEncoder(metrics)
	con.SetWriteDeadline(timeout)
	con.SetReadDeadline(timeout)

	failfunc := func(err error) (*Peer, []Endpoint, error) {
		tmplogger.WithError(err).Debug("Handshake failed")
		con.Close()
		return nil, nil, err
	}

	err := encoder.Encode(handshake)
	if err != nil {
		return failfunc(fmt.Errorf("Failed to send handshake"))
	}

	var reply Handshake
	err = decoder.Decode(&reply)
	if err != nil {
		return failfunc(fmt.Errorf("Failed to read handshake"))
	}

	// check basic structure
	if err = reply.Valid(c.net.conf); err != nil {
		return failfunc(err)
	}

	endpoint, err := NewEndpoint(ip, reply.Header.PeerPort)
	if err != nil {
		return failfunc(fmt.Errorf("failed to create endpoint: %v", err))
	}

	if c.isBannedEndpoint(endpoint) {
		return failfunc(fmt.Errorf("Peer %s is banned, disconnecting", endpoint))
	}

	// loopback detection
	if bytes.Equal(reply.Payload, nonce) {
		return failfunc(fmt.Errorf("loopback"))
	}

	prot, err := c.determineProtocol(&reply, con, decoder, encoder)
	if err != nil {
		return failfunc(err)
	}

	// dialed a node that's full
	if reply.Header.Type == TypeRejectAlternative {
		con.Close()
		tmplogger.Debug("con rejected with alternatives")

		share, err := prot.ParsePeerShare(reply.Payload)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse alternatives: %s", err.Error())
		}

		for _, ep := range share {
			if !ep.Valid() {
				return nil, nil, fmt.Errorf("peer provided invalid peer share")
			}
		}
		return nil, share, fmt.Errorf("connection rejected")
	}

	peer := newPeer(c.net, uint32(reply.Header.NodeID), endpoint, con, prot, metrics, incoming)

	c.peerStatus <- peerStatus{peer: peer, online: true}

	return peer, nil, nil
}

func (c *controller) determineProtocol(hs *Handshake, conn net.Conn, decoder *gob.Decoder, encoder *gob.Encoder) (Protocol, error) {
	v := hs.Header.Version
	if v > c.net.conf.ProtocolVersion {
		v = c.net.conf.ProtocolVersion
	}

	switch v {
	case 9:
		v9 := new(ProtocolV9)
		v9.init(c.net, decoder, encoder)
		return v9, nil
	case 10:
		v10 := new(ProtocolV10)
		v10.init(decoder, encoder)
		return v10, nil
	default:
		return nil, fmt.Errorf("unknown protocol version %d", v)
	}
}

// RejectWithShare rejects an incoming connection by sending them a handshake that provides
// them with alternative peers to connect to
func (c *controller) RejectWithShare(con net.Conn, share []Endpoint) error {
	defer con.Close() // we're rejecting, so always close

	payload, err := json.Marshal(share)
	if err != nil {
		return err
	}

	handshake := newHandshake(c.net.conf, payload)
	handshake.Header.Type = TypeRejectAlternative

	// only push the handshake, don't care what they send us
	encoder := gob.NewEncoder(con)
	con.SetWriteDeadline(time.Now().Add(c.net.conf.HandshakeTimeout))
	err = encoder.Encode(handshake)
	if err != nil {
		return err
	}

	return nil
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

	peer, alternatives, err := c.handshake(ep.IP, con, false)
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

	c.logger.Debugf("Handshake success for peer %s, version %s", peer.Hash, peer.prot.Version())
	return true, nil
}

// listen listens for incoming TCP connections and passes them off to handshake maneuver
func (c *controller) listen() {
	tmpLogger := c.logger.WithFields(log.Fields{"address": c.net.conf.BindIP, "port": c.net.conf.ListenPort})
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

		go c.handleIncoming(conn)
	}
}
