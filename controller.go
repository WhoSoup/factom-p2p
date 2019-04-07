// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

// controller manages the P2P Network.
// It maintains the list of peers, and has a master run-loop that
// processes ingoing and outgoing messages.
// It is controlled via a command channel.
// Other than Init and NetworkStart, all administration is done via the channel.

import (
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
)

var controllerLogger = packageLogger.WithField("subpack", "controller")

// controller manages the peer to peer network.
type controller struct {
	net     *Network
	running bool

	listenPort string // port to listen on for new connections

	In   chan interface{} // incoming messages from the network to the app
	Out  chan interface{} // outgoing messages to the network
	stop chan interface{}

	ToNetwork   chan interface{} // Parcels from the application for us to route
	FromNetwork chan interface{} // Parcels from the network for the application

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
	go c.listenLoop()
	c.routeLoop()
}

// Stop shuts down the peer manager and all active connections
func (c *controller) Stop() {
	c.stop <- true
}

// listenLoop listens for incoming TCP connections and passes them off to peer manager
func (c *controller) listenLoop() {
	c.logger.Debug("controller.listenLoop() starting up")

	tmpLogger := c.logger.WithFields(log.Fields{"address": c.net.conf.ListenIP, "port": c.net.conf.ListenPort})

	addr := fmt.Sprintf("%s:%s", c.net.conf.ListenIP, c.net.conf.ListenPort)
	fmt.Println("addr:", addr, c.net.conf.ListenIP, c.net.conf.ListenPort)
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

// Take messages from Outgoing channel and route it to the peer manager
func (c *controller) routeLoop() {
	c.logger.Debugf("controller.routeLoop() @@@@@@@@@@ starting up in %d seconds", 2)
	time.Sleep(time.Second * time.Duration(2)) // Wait a few seconds to let the system come up.

	// terminates via stop channel
	for {
		// TODO move this to peer manager
		//c.connections.UpdatePrometheusMetrics()
		//p2pcontrollerNumMetrics.Set(float64(len(c.connectionMetrics)))

		// blocking read on c.Out, and c.stop
		select {
		case message := <-c.Out:
			c.handleParcel(message)
		// stop this loop if anything shows up
		case <-c.stop:
			return
		}
	}
}

func (c *controller) handleParcel(message interface{}) {
	parcel, ok := message.(*Parcel)
	if !ok {
		c.logger.WithField("message", message).Errorf("handleParcel() received unexpected message of type %s", reflect.TypeOf(message))
		return
	}
	TotalMessagesSent++

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
