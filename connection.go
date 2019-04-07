// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"encoding/gob"
	"net"
	"time"

	"github.com/FactomProject/factomd/common/primitives"

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

	Send    ParcelChannel // messages from the other side
	Receive ParcelChannel // messages to the other side
	Error   chan error    // connection died

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

// Each connection is a simple state machine.  The state is managed by a single goroutine which also does networking.
// The flow is this:  Connection gets initialized, and either has a peer or a net connection (From an accept())
// If no network connection, the Connection dials.  If the dial is successful, it moves to the Online state
// If not, it moves to the Shutdown state-- we only dial out once when initialized with a peer.
// If we are online and get a network error, we shift to offline mode.  In offline state we attempt to reconnect for
// a period defined in protocol.go.  IF successful, we go back Online.  If too many attempts are made, we go to
// The ConnectionShutdown state, and exit the runloop.  In the Shutdown state we notify the controller so that we can be
// cleaned up.
const (
	ConnectionInitialized  uint8 = iota //Structure created, have peer info. Dial command moves us to Online or Shutdown (depending)
	ConnectionOnline                    // We're connected to the other side.  Normal state
	ConnectionOffline                   // We've been disconnected for whatever reason.  Attempt to reconnect some number of times. Moves to Online if successful, Shutdown if not.
	ConnectionShuttingDown              // We're shutting down, the receives loop exits.
	ConnectionClosed                    // We're shut down, the runloop sets this state right before exiting. Controller can clean us up.
)

// Map of network ids to strings for easy printing of network ID
var connectionStateStrings = map[uint8]string{
	ConnectionInitialized:  "Initialized",
	ConnectionOnline:       "Online",
	ConnectionOffline:      "Offline",
	ConnectionShuttingDown: "Shutting Down",
	ConnectionClosed:       "Closed",
}

type GracefulShutdown struct {
}

func (g *GracefulShutdown) Error() string {
	return "Graceful Shutdown initiated"
}

// ConnectionParcel is sent to convey an application message destined for the network.
type ConnectionParcel struct {
	Parcel Parcel
}

func (e *ConnectionParcel) JSONByte() ([]byte, error) {
	return primitives.EncodeJSON(e)
}

func (e *ConnectionParcel) JSONString() (string, error) {
	return primitives.EncodeJSONString(e)
}

func (e *ConnectionParcel) String() string {
	str, _ := e.JSONString()
	return str
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

// ConnectionCommand is used to instruct the Connection to carry out some functionality.
type ConnectionCommand struct {
	Command uint8
	Peer    Peer
	Delta   int32
	Metrics ConnectionMetrics
}

func (e *ConnectionCommand) JSONByte() ([]byte, error) {
	return primitives.EncodeJSON(e)
}

func (e *ConnectionCommand) JSONString() (string, error) {
	return primitives.EncodeJSONString(e)
}

func (e *ConnectionCommand) String() string {
	str, _ := e.JSONString()
	return str
}

// These are the commands that connections can send/receive
const (
	ConnectionIsClosed uint8 = iota // Notifies the controller that we are shut down and can be released
	ConnectionShutdownNow
	ConnectionUpdatingPeer
	ConnectionAdjustPeerQuality
	ConnectionUpdateMetrics
	ConnectionGoOffline // Notifies the connection it should go offinline (eg from another goroutine)
)

//////////////////////////////
//
// Public API
//
//////////////////////////////

func NewConnection(peerHash string, conn net.Conn, receive ParcelChannel, net *Network) *Connection {
	c := &Connection{}
	c.Send = NewParcelChannel(net.conf.ChannelCapacity)
	c.Receive = NewParcelChannel(net.conf.ChannelCapacity)
	c.Error = make(chan error, 3) // two goroutines + close() = max 3 errors

	c.logger = conLogger.WithFields(log.Fields{"address": conn.RemoteAddr(), "peer": peerHash})
	c.logger.Debug("Connection initialized")

	c.readDeadline = net.conf.ReadDeadline
	c.writeDeadline = net.conf.WriteDeadline

	c.conn = conn
	c.encoder = gob.NewEncoder(c.conn)
	c.decoder = gob.NewDecoder(c.conn)

	return c
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
		parcel := <-c.Send.Reader()

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

		c.metrics.BytesSent += parcel.Header.Length
		c.metrics.MessagesSent++
		c.LastSend = time.Now()
	}
}

func (c *Connection) Stop() {
	c.logger.Debug("Stopping connection")
	c.Error <- &GracefulShutdown{}
	c.conn.Close() // this will force both sendLoop and readLoop to stop immediately
}
