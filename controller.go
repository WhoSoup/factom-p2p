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
	net      *Network
	stop     chan bool
	listener *LimitedListener

	logger *log.Entry
}

// newcontroller creates a new P2P Network controller with the specified configuration
func newController(network *Network) *controller {
	c := &controller{}
	c.logger = controllerLogger.WithFields(log.Fields{
		"node":    network.conf.NodeName,
		"port":    network.conf.ListenPort,
		"network": fmt.Sprintf("%#x", network.conf.Network)})
	c.net = network
	return c
}

func (c *controller) Start() {

}

// Stop shuts down the peer manager and all active connections
func (c *controller) Stop() {

}
