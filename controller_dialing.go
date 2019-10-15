package p2p

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// CAT responsible for filling connections back up to conf.Target connections
func (c *controller) dialLoop() {
	c.logger.Debug("Start dialLoop()")
	defer c.logger.Debug("Stop dialLoop()")

	for {
		select {
		case ip := <-c.dial:
			total := c.peers.Total()
			if uint(total) >= c.net.conf.Target { // this will clear c.dial if target reached
				break
			}

			if c.peers.IsConnected(ip.Address) {

			}
		}
	}
}

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
					c.peers.Remove(old)
				}
				err := c.peers.Add(pc.peer)
				if err != nil {
					c.logger.Errorf("Unable to add peer %s to peer store because an old peer still exists", pc.peer)
				}
			} else {
				c.peers.Remove(pc.peer)
				if pc.peer.IsIncoming {
					// lock this connection temporarily so we don't try to connect to it
					// before they can reconnect
				}
			}
			if c.net.prom != nil {
				c.net.prom.Connections.Set(float64(c.peers.Total()))
				c.net.prom.Unique.Set(float64(c.peers.Unique()))
				c.net.prom.Incoming.Set(float64(c.peers.Incoming()))
				c.net.prom.Outgoing.Set(float64(c.peers.Outgoing()))
			}
		}
	}
}
func (c *controller) allowIncoming(addr string) error {
	if c.isBannedAddress(addr) {
		return fmt.Errorf("Address %s is banned", addr)
	}

	if uint(c.peers.Total()) >= c.net.conf.Incoming && !c.isSpecialAddr(addr) {
		return fmt.Errorf("Refusing incoming connection from %s because we are maxed out (%d of %d)", addr, c.peers.Total(), c.net.conf.Incoming)
	}

	if c.net.conf.PeerIPLimitIncoming > 0 && uint(c.peers.Count(addr)) >= c.net.conf.PeerIPLimitIncoming {
		return fmt.Errorf("Rejecting %s due to per ip limit of %d", addr, c.net.conf.PeerIPLimitIncoming)
	}

	return nil
}

func (c *controller) handleIncoming(con net.Conn) {
	if c.net.prom != nil {
		c.net.prom.Connecting.Inc()
		defer c.net.prom.Connecting.Dec()
	}

	addr, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		c.logger.WithError(err).Debugf("Unable to parse address %s", con.RemoteAddr().String())
		con.Close()
		return
	}

	// port is overriden during handshake, use default port as temp port
	ip, err := NewIP(addr, c.net.conf.ListenPort)
	if err != nil { // should never happen for incoming
		c.logger.WithError(err).Debugf("Unable to decode address %s", addr)
		con.Close()
		return
	}

	if err = c.allowIncoming(addr); err != nil {
		c.logger.WithError(err).Infof("Rejecting connection")
		con.Close()
		return
	}

	peer := newPeer(c.net, c.peerStatus, c.peerData)
	if ok, err := peer.StartWithHandshake(ip, con, true); ok {
		c.logger.Debugf("Incoming handshake success for peer %s, version %s", peer.Hash, peer.prot.Version())

		if c.isBannedIP(peer.IP) {
			c.logger.Debugf("Peer %s is banned, disconnecting", peer.Hash)
			return
		}
		c.dialer.Reset(peer.IP)
	} else {
		c.logger.WithError(err).Debugf("Handshake failed for address %s, stopping", ip)
		peer.Stop()
	}
}

func (c *controller) Dial(ip IP) {
	if c.net.prom != nil {
		c.net.prom.Connecting.Inc()
		defer c.net.prom.Connecting.Dec()
	}

	if ip.Port == "" {
		ip.Port = c.net.conf.ListenPort // TODO add a "default port"?
		c.logger.Debugf("Dialing to %s (with no previously known port)", ip)
	} else {
		c.logger.Debugf("Dialing to %s", ip)
	}

	con, err := c.dialer.Dial(ip)
	if err != nil {
		c.logger.WithError(err).Infof("Failed to dial to %s", ip)
		return
	}

	peer := newPeer(c.net, c.peerStatus, c.peerData)
	if ok, err := peer.StartWithHandshake(ip, con, false); ok {
		c.logger.Debugf("Handshake success for peer %s, version %s", peer.Hash, peer.prot.Version())
		c.dialer.Reset(peer.IP)
	} else if err.Error() == "loopback" {
		c.logger.Debugf("Banning ourselves for 50 years")
		c.banIP(ip, time.Hour*24*365*50) // ban for 50 years
		peer.Stop()
	} else {
		c.logger.WithError(err).Debugf("Handshake fail with %s", ip)
		peer.Stop()
	}
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
					return
				}
			}
			continue
		}

		go c.handleIncoming(conn)
	}
}
