// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

// Controller manages the P2P Network.
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

// packageLogger is the general logger for all p2p related logs. You can add additional fields,
// or create more context loggers off of this
var packageLogger = log.WithFields(log.Fields{
	"package":   "p2p",
	"component": "networking"})

var controllerLogger = packageLogger.WithField("subpack", "controller")

// Controller manages the peer to peer network.
type Controller struct {
	running bool

	listenPort string // port to listen on for new connections

	peerManager *PeerManager
	Config      *Configuration
	In          chan interface{} // incoming messages from the network to the app
	Out         chan interface{} // outgoing messages to the network
	stop        chan interface{}

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

// NewController creates a new P2P Network controller with the specified configuration
func NewController(config *Configuration) *Controller {
	c := &Controller{}
	c.logger = controllerLogger.WithFields(log.Fields{
		"node":    config.NodeName,
		"port":    config.ListenPort,
		"network": fmt.Sprintf("%#x", config.Network)})
	c.Config = config
	c.peerManager = NewPeerManager(c)

	// TODO initialize channels
	c.stop = make(chan interface{}, 1)                  // controller -> self
	c.In = make(chan interface{}, StandardChannelSize)  // network -> app
	c.Out = make(chan interface{}, StandardChannelSize) // app -> network

	return c
}

// Start initializes the network by starting the peer manager and listening to incoming connections
func (c *Controller) Start() {
	c.logger.Info("Starting the P2P Network")

	go c.peerManager.Start() // this will get peer manager ready to handle incoming connections

	go c.listenLoop()
	go c.routeLoop()
}

func (c *Controller) DebugMessage() string {
	out := fmt.Sprintf("Controller for Node #%d: %s:%s\n", c.Config.NodeID, c.Config.ListenIP, c.Config.ListenPort)
	for _, p := range c.peerManager.peers.Slice() {
		out += fmt.Sprintf("\tPeer #%s\n", p.Hash)
		out += fmt.Sprintf("\t\tState: %s\n", p.state)
	}
	return out
}

// Stop shuts down the peer manager and all active connections
func (c *Controller) Stop() {
	c.running = false
	c.stop <- true
	c.peerManager.Stop()
}

func (c *Controller) CreateParcel(pType ParcelCommandType, payload []byte) *Parcel {
	parcel := NewParcel(pType, payload)
	return parcel
}

// listenLoop listens for incoming TCP connections and passes them off to peer manager
func (c *Controller) listenLoop() {
	c.logger.Debug("Controller.listenLoop() starting up")

	tmpLogger := c.logger.WithFields(log.Fields{"address": c.Config.ListenIP, "port": c.Config.ListenPort})

	addr := fmt.Sprintf("%s:%s", c.Config.ListenIP, c.Config.ListenPort)
	fmt.Println("addr:", addr, c.Config.ListenIP, c.Config.ListenPort)
	listener, err := NewLimitedListener(addr, c.Config.ListenLimit)
	if err != nil {
		tmpLogger.WithError(err).Error("Controller.Start() unable to start limited listener")
		return
	}
	defer listener.Close()

	tmpLogger.Info("Listening for new connections")

	// start permanent loop
	// terminates on program exit
	for {
		conn, err := listener.Accept()
		if err != nil {
			tmpLogger.WithError(err).Warn("Controller.acceptLoop() error accepting")
			continue
		}

		// currently a non-concurrent implementation
		c.peerManager.HandleIncoming(conn)
	}
}

// Take messages from Outgoing channel and route it to the peer manager
func (c *Controller) routeLoop() {
	c.logger.Debugf("Controller.routeLoop() @@@@@@@@@@ starting up in %d seconds", 2)
	time.Sleep(time.Second * time.Duration(2)) // Wait a few seconds to let the system come up.

	// terminates via stop channel
	for {
		// TODO move this to peer manager
		//c.connections.UpdatePrometheusMetrics()
		//p2pControllerNumMetrics.Set(float64(len(c.connectionMetrics)))

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

func (c *Controller) handleParcel(message interface{}) {
	parcel, ok := message.(*Parcel)
	if !ok {
		c.logger.WithField("message", message).Errorf("handleParcel() received unexpected message of type %s", reflect.TypeOf(message))
		return
	}
	TotalMessagesSent++

	switch parcel.Header.TargetPeer {
	case FullBroadcastFlag:
		c.peerManager.Broadcast(parcel, true)
	case BroadcastFlag:
		c.peerManager.Broadcast(parcel, true)
	case RandomPeerFlag:
		c.peerManager.ToPeer("", parcel)
	default:
		c.peerManager.ToPeer(parcel.Header.TargetPeer, parcel)
	}
}
