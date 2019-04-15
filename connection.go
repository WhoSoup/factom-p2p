// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"encoding/gob"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// conLogger is the general logger for all connection related logs. You can add additional fields,
// or create more context loggers off of this
var conLogger = packageLogger.WithField("subpack", "connection")

// Connection represents a single connection to another peer over the network. It communicates with the application
// via two channels, send and receive.  These channels take structs of type ConnectionCommand or ConnectionParcel
// (defined below).
type Connection struct {
	conn net.Conn

	Socket   string
	Address  string
	stop     chan bool
	Send     ParcelChannel // messages from the other side
	Receive  ParcelChannel // messages to the other side
	Error    chan error    // connection died
	Outgoing bool

	NodeID uint64

	writeDeadline time.Duration
	readDeadline  time.Duration
	encoder       *gob.Encoder // Wire format is gobs in this version, may switch to binary
	decoder       *gob.Decoder // Wire format is gobs in this version, may switch to binary

	LastRead time.Time
	LastSend time.Time

	// and as "address" for sending messages to specific nodes.
	metrics ConnectionMetrics // Metrics about this connection

	// logging
	logger *log.Entry
}

type GracefulShutdown struct {
}

func (g *GracefulShutdown) Error() string {
	return "Graceful Shutdown initiated"
}

// ConnectionMetrics is used to encapsulate various metrics about the connection.
type ConnectionMetrics struct {
	MomentConnected  time.Time // when the connection started.
	BytesSent        uint32    // Keeping track of the data sent/received for console
	BytesReceived    uint32    // Keeping track of the data sent/received for console
	MessagesSent     uint32    // Keeping track of the data sent/received for console
	MessagesReceived uint32    // Keeping track of the data sent/received for console
	PeerAddress      string    // Peer IP Address
	PeerQuality      int32     // Quality of the connection.
	PeerType         string    // Type of the peer (regular, special_config, ...)
	// Red: Below -50
	// Yellow: -50 - 100
	// Green: > 100
	ConnectionState string // Basic state of the connection
	ConnectionNotes string // Connectivity notes for the connection
}

func NewConnection(n *Network, address string, conn net.Conn, receive ParcelChannel, outgoing bool) *Connection {
	c := &Connection{}
	c.Send = NewParcelChannel(n.conf.ChannelCapacity)
	c.Receive = receive
	c.Error = make(chan error, 3) // two goroutines + close() = max 3 errors
	c.stop = make(chan bool, 5)

	c.Address = address
	c.Socket = conn.RemoteAddr().String()

	c.logger = conLogger.WithFields(log.Fields{"address": conn.RemoteAddr().String(), "node": n.conf.NodeName})
	c.logger.Debug("Connection initialized")

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
			c.logger.WithError(err).Debug("Terminating readLoop because of error")
			return
		}

		c.metrics.BytesReceived += message.Header.Length
		c.metrics.MessagesReceived++
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
				c.logger.WithError(err).Debug("Terminating sendLoop because of error")
				return
			}
			c.logger.Debug("parcel sent")
			c.metrics.BytesSent += parcel.Header.Length
			c.metrics.MessagesSent++
			c.LastSend = time.Now()
		}
	}
}

func (c *Connection) Stop() {
	c.logger.Debug("Stopping connection")
	c.Error <- &GracefulShutdown{}
	c.stop <- true // this will stop sendloop
	c.conn.Close() // this will stop readloop
}
