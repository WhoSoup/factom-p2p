// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var peerLogger = packageLogger.WithField("subpack", "peer")

// Data structures and functions related to peers (eg other nodes in the network)

type PeerType uint8

const (
	RegularPeer        PeerType = iota
	SpecialPeerConfig           // special peer defined in the config file
	SpecialPeerCmdLine          // special peer defined via the cmd line params
	PeerIncoming
	PeerOutgoing
)

// PeerState is the states for the Peer's state machine
type PeerState uint8

func (ps PeerState) String() string {
	switch ps {
	case Offline:
		return "Offline"
	case Connecting:
		return "Connecting"
	case Online:
		return "Online"
	case Uninitialized:
		return "Uninitialized"
	default:
		return "Unknown state"
	}
}

// The peer state machine's states
const (
	Uninitialized PeerState = iota
	Offline
	Connecting
	Online
)

// Peer is a representation of another node in the network
//
// Peers exist in two forms: Temporary peers and regular peers
type Peer struct {
	net *Network

	// connection related
	connChannel chan *Connection
	conn        *Connection
	connError   chan error
	connMutex   sync.RWMutex

	// current state
	state      PeerState
	IsIncoming bool

	//	stateMutex             sync.RWMutex
	age  time.Time
	stop chan bool

	lastPeerRequest        time.Time
	lastPeerSend           time.Time
	Receive                ParcelChannel
	connectionAttempt      time.Time
	connectionAttemptCount uint
	LastReceive            time.Time // Keep track of how long ago we talked to the peer.
	LastSend               time.Time // Keep track of how long ago we talked to the peer.

	Port      string
	Temporary bool
	Seed      bool
	Dialable  bool

	QualityScore int32  // 0 is neutral quality, negative is a bad peer.
	Address      string // Must be in form of x.x.x.x
	NodeID       uint64 // a nonce to distinguish multiple nodes behind one IP address
	Hash         string // This is more of a connection ID than hash right now.
	Location     uint32 // IP address as an int.
	Type         PeerType

	// logging
	logger *log.Entry
}

func NewPeer(net *Network, address string) *Peer {
	p := &Peer{Address: address, state: Uninitialized}
	p.net = net
	p.logger = peerLogger.WithFields(log.Fields{
		"node":    net.conf.NodeName,
		"hash":    p.Hash,
		"address": p.Address,
		"Port":    p.Port,
	})
	p.age = time.Now()
	p.stop = make(chan bool, 1)
	p.connChannel = make(chan *Connection, 10)
	p.Receive = NewParcelChannel(net.conf.ChannelCapacity)
	p.Hash = fmt.Sprintf("%x", net.rng.Int63())
	p.logger.Debugf("Creating new peer")
	return p
}

func (p *Peer) Start() {
	if p.state != Uninitialized {
		p.logger.Error("Tried to Start a peer that is already initialized")
		return
	}

	p.connMutex.Lock()
	if p.conn != nil {
		p.conn.Stop()
	}
	p.conn = nil
	p.state = Offline
	p.connError = nil
	p.connMutex.Unlock()
	p.connectionAttemptCount = 0

	go p.monitorConnection()
}

// Stop disconnects the peer from its active connection
//
// A hard stop will also disable the peer's goroutines, preparing it for deletion
func (p *Peer) Stop() {
	p.stop <- true
}

func (p *Peer) setConnection(c *Connection) {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()
	if p.conn != nil { // stop active connection
		defer func(old *Connection) {
			old.Stop()
		}(p.conn) // pass as value
	}
	p.conn = nil
	p.connError = nil

	for len(p.Receive) > 0 { // drop old messages
		<-p.Receive
	}
	if c == nil { // go offline
		p.logger.Debug("Going offline")
		p.state = Offline
	} else { // handle a new connection
		p.logger.Debugf("Accepting connection %s", c.conn.RemoteAddr().String())
		p.state = Online
		p.conn = c
		p.connError = c.Error
		p.connectionAttemptCount = 0
	}
}

// monitorConnection watches the underlying Connection and the connection channel
//
// Any data arriving via connection will be passed on to the peer manager.
// If the connection dies, change state to Offline. If a new connection arrives, handle it.
func (p *Peer) monitorConnection() {
	for {
		select {
		case <-p.stop: // tear down the peer completely
			p.logger.Debug("Connection shutting down")
			p.setConnection(nil)
			return
		case err := <-p.connError: // if an error arrives here, the connection already stops itself
			p.logger.WithError(err).Debug("Connection error")
			p.setConnection(nil)
		case c := <-p.connChannel:
			p.setConnection(c)
		case parcel := <-p.Receive:
			p.QualityScore++
			p.logger.Debugf("Received incoming parcel: %v", parcel)
			if newport := parcel.Header.PeerPort; newport != p.Port {
				p.logger.WithFields(log.Fields{"old": p.Port, "new": newport}).Debugf("Listen port changed")
				p.Port = newport
			}

			if nodeid := parcel.Header.NodeID; nodeid != p.NodeID {
				p.logger.WithFields(log.Fields{"old": p.NodeID, "new": nodeid}).Debugf("NodeID changed")
				p.NodeID = nodeid
			}
			p.LastReceive = time.Now()
			select {
			case p.net.peerParcel <- PeerParcel{Peer: p, Parcel: parcel}:
			default:
				p.logger.Warn("Peer manager unable to handle load")
			}
		}
	}
}

func (p *Peer) String() string {
	return fmt.Sprintf("%s %s:%s", p.Hash, p.Address, p.Port)
}

func (p *Peer) ConnectAddress() string {
	return fmt.Sprintf("%s:%s", p.Address, p.Port)
}

func (p *Peer) PeerShare() PeerShare {
	return PeerShare{
		Address:      p.Address,
		Port:         p.Port,
		QualityScore: p.QualityScore}
}

func (p *Peer) CanUpgrade() bool {
	return p.Temporary && p.NodeID != 0
}

func (p *Peer) StartToDial() {
	go p.dialInternal()
}

func (p *Peer) dialInternal() {
	if !p.CanDial() {
		p.logger.Errorf("Maximum dial attempts reached")
		return
	}
	if p.state == Connecting {
		p.logger.Debug("Still attempting to connect")
		return
	}

	var newcon *Connection
	defer func() { // update peer after dialing is over
		// will go offline if newcon ends up being nil
		p.connChannel <- newcon
	}()

	p.connMutex.Lock()
	defer p.connMutex.Unlock() // executes before the above setConnection

	p.state = Connecting
	p.connectionAttempt = time.Now()
	p.connectionAttemptCount++
	p.logger.WithField("attempt", p.connectionAttemptCount).Debugf("Dialing to %s:%s", p.Address, p.Port)

	if p.Location == 0 {
		loc, err := IP2Location(p.Address)
		if err != nil {
			p.logger.WithError(err).Warnf("Unable to convert address %s to location", p.Address)
			return
		}
		p.Location = loc
	}

	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", p.net.conf.BindIP))
	if err != nil {
		p.logger.WithError(err).Errorf("Unable to resolve local interface \"%s:0\"", p.net.conf.BindIP)
	}

	dialer := net.Dialer{
		LocalAddr: local,
		Timeout:   p.net.conf.DialTimeout,
	}

	p.state = Connecting
	con, err := dialer.Dial("tcp", fmt.Sprintf("%s:%s", p.Address, p.Port))
	//con, err := net.DialTCP("tcp", local, remote)

	if err != nil {
		p.logger.WithError(err).Infof("Unable to connect to peer")
		return
	}

	newcon = NewConnection(p.net, p.Address, con, p.Receive, true)
	newcon.Start()

	// newcon will update through defer
}

func (p *Peer) HandleActiveConnection(con *Connection) {
	p.connChannel <- con
}

func (p *Peer) HandleActiveTCP(con net.Conn) {

	//newcon := NewConnection(p.Hash, con, p.Receive, p.net, false)
	//newcon.Start()
	//p.connChannel <- newcon
}

func (p *Peer) Send(parcel *Parcel) {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()
	if p.conn == nil || p.state != Online {
		if p.state == Connecting {
			log.Error("Tried to send parcel connection still connecting")
		} else {
			log.Error("Tried to send parcel on offline connection")
		}
		return
	}
	// TODO check peer state machine

	p.LastSend = time.Now()
	// send this parcel from this peer
	parcel.Header.Network = p.net.conf.Network
	parcel.Header.Version = p.net.conf.ProtocolVersion
	parcel.Header.NodeID = p.net.conf.NodeID
	parcel.Header.PeerPort = string(p.net.conf.ListenPort) // notify other side of our port
	p.conn.Send.Send(parcel)
}

func (p *Peer) CanDial() bool {
	return p.connectionAttemptCount < p.net.conf.RedialAttempts && p.Port != "0"
}

func (p *Peer) IsOnline() bool {
	return p.state == Online
}
func (p *Peer) IsOffline() bool {
	return p.state == Offline
}

func (p *Peer) Shareable() bool {
	return p.QualityScore >= p.net.conf.MinimumQualityScore && !p.Temporary
}

func (p *Peer) PeerLogFields() log.Fields {
	return log.Fields{
		"address": p.Address,
		"port":    p.Port,
		//"peer_type": p.PeerTypeString(),
	}
}

// PeerQualitySort sorts peers by quality score, ascending
type PeerQualitySort []Peer

func (p PeerQualitySort) Len() int {
	return len(p)
}
func (p PeerQualitySort) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p PeerQualitySort) Less(i, j int) bool {
	return p[i].QualityScore < p[j].QualityScore
}
