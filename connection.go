// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// conLogger is the general logger for all connection related logs. You can add additional fields,
// or create more context loggers off of this
var conLogger = packageLogger.WithField("subpack", "connection")

// Connection is a simple TCP wrapper, sending data from Send and reading to Receive
type Connection struct {
	conn net.Conn

	stop    chan bool
	Send    ParcelChannel // messages from the other side
	Receive ParcelChannel // messages to the other side
	Error   chan error    // connection died

	writeDeadline time.Duration
	readDeadline  time.Duration
	encoder       *gob.Encoder // Wire format is gobs in this version, may switch to binary
	decoder       *gob.Decoder // Wire format is gobs in this version, may switch to binary

	LastRead time.Time
	LastSend time.Time

	// logging
	logger *log.Entry
}

func NewConnection(n *Network, conn net.Conn, send, receive ParcelChannel) *Connection {
	c := &Connection{}
	c.logger = conLogger.WithFields(log.Fields{"address": conn.RemoteAddr().String(), "node": n.conf.NodeName})
	c.logger.Debug("Connection initialized")

	c.Send = send
	c.Receive = receive
	c.Error = make(chan error, 3) // two goroutines + close() = max 3 errors
	c.stop = make(chan bool, 5)
	c.readDeadline = n.conf.ReadDeadline
	c.writeDeadline = n.conf.WriteDeadline
	c.conn = conn
	c.encoder = gob.NewEncoder(c.conn)
	c.decoder = gob.NewDecoder(c.conn)
	return c
}

func (c *Connection) String() string {
	return c.conn.RemoteAddr().String()
}

// Start the connection, make it read and write from the connection
// starts two goroutines
func (c *Connection) Start() {
	c.logger.Debug("Starting connection")
	go c.readLoop()
	go c.sendLoop()
}

func (c *Connection) readLoop() {
	defer c.conn.Close() // close connection on fatal error
	for {
		var message Parcel

		c.conn.SetReadDeadline(time.Now().Add(c.readDeadline))
		err := c.decoder.Decode(&message)
		if err != nil {
			c.Error <- err
			return
		}

		c.LastRead = time.Now()
		c.Receive.Send(&message)
	}
}

// sendLoop listens to the Outgoing channel, pushing all data from there
// to the tcp connection
func (c *Connection) sendLoop() {
	defer c.conn.Close() // close connection on fatal error
	for {
		select {
		case <-c.stop:
			return
		case parcel := <-c.Send:
			if parcel == nil {
				c.logger.Error("Received <nil> pointer")
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline))
			err := c.encoder.Encode(parcel)
			if err != nil { // no error is recoverable
				c.Error <- err
				return
			}
			c.LastSend = time.Now()
		}
	}
}

func (c *Connection) Stop() {
	c.logger.Debug("Stopping connection")
	c.Error <- fmt.Errorf("Manually initialized shutdown")
	c.stop <- true // this will stop sendloop
	//c.conn.Close() // this will stop readloop
}
