// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

var controllerLogger = packageLogger.WithField("subpack", "controller")

// controller manages the peer to peer network.
type controller struct {
	net  *Network
	stop chan interface{}

	logger *log.Entry
}

//////////////////////////////////////////////////////////////////////
// Public (exported) methods.
//
// The surface for interfacting with this is very minimal to avoid deadlocks
// and allow maximum concurrency.
// Other than setup, these API communicate with the controller via the
// command channel.
//////////////////////////////////////////////////////////////////////

// newcontroller creates a new P2P Network controller with the specified configuration
func newController(network *Network) *controller {
	c := &controller{}
	c.logger = controllerLogger.WithFields(log.Fields{
		"node":    network.conf.NodeName,
		"port":    network.conf.ListenPort,
		"network": fmt.Sprintf("%#x", network.conf.Network)})
	c.net = network
	c.stop = make(chan interface{}, 1) // controller -> self
	return c
}

func (c *controller) Start() {
	for len(c.stop) > 0 {
		<-c.stop
	}
	go c.listenLoop()
	c.routeLoop()
}

// Stop shuts down the peer manager and all active connections
func (c *controller) Stop() {
	c.stop <- true
}

// listenLoop listens for incoming TCP connections and passes them off to peer manager
func (c *controller) listenLoop() {
	tmpLogger := c.logger.WithFields(log.Fields{"address": c.net.conf.ListenIP, "port": c.net.conf.ListenPort})
	tmpLogger.Debug("controller.listenLoop() starting up")

	addr := fmt.Sprintf("%s:%s", c.net.conf.ListenIP, c.net.conf.ListenPort)

	listener, err := NewLimitedListener(addr, c.net.conf.ListenLimit)
	if err != nil {
		tmpLogger.WithError(err).Error("controller.Start() unable to start limited listener")
		return
	}
	defer listener.Close()

	tmpLogger.Info("Listening for new connections")

	// start permanent loop
	// terminates on program exit
	for {
		conn, err := listener.Accept()
		if err != nil {
			tmpLogger.WithError(err).Warn("controller.acceptLoop() error accepting")
			continue
		}

		// currently a non-concurrent implementation
		c.net.peerManager.HandleIncoming(conn)
	}
}

// routLoop Takes messages from the network's ToNetwork channel and routes it
// to the peerManager via the appropriate function
func (c *controller) routeLoop() {
	for {
		// TODO metrics?
		// blocking read on ToNetwork, and c.stop
		select {
		case message := <-c.net.ToNetwork:
			c.handleParcel(message)
		// stop this loop if anything shows up
		case <-c.stop:
			return
		}
	}
}

func (c *controller) handleParcel(parcel *Parcel) {
	switch parcel.Header.TargetPeer {
	case FullBroadcastFlag:
		c.net.peerManager.Broadcast(parcel, true)
	case BroadcastFlag:
		c.net.peerManager.Broadcast(parcel, true)
	case RandomPeerFlag:
		c.net.peerManager.ToPeer("", parcel)
	default:
		c.net.peerManager.ToPeer(parcel.Header.TargetPeer, parcel)
	}
}
