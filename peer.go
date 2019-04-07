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
	default:
		return "Unknown state"
	}
}

// The peer state machine's states
const (
	Offline PeerState = iota
	Connecting
	Online
)

type Peer struct {
	net       *Network
	conn      *Connection
	connMutex sync.RWMutex
	state     PeerState
	//	stateMutex             sync.RWMutex
	age                    time.Time
	stop                   chan interface{}
	IsOutgoing             bool
	config                 *Configuration
	lastPeerRequest        time.Time
	lastPeerSend           time.Time
	incoming               chan *Parcel
	Receive                ParcelChannel
	connectionAttempt      time.Time
	connectionAttemptCount uint

	ListenPort string

	QualityScore int32     // 0 is neutral quality, negative is a bad peer.
	Address      string    // Must be in form of x.x.x.x
	Port         string    // Must be in form of xxxx
	NodeID       uint64    // a nonce to distinguish multiple nodes behind one IP address
	Hash         string    // This is more of a connection ID than hash right now.
	Location     uint32    // IP address as an int.
	Network      NetworkID // The network this peer reference lives on.
	Type         PeerType
	Connections  int                  // Number of successful connections.
	LastContact  time.Time            // Keep track of how long ago we talked to the peer.
	Source       map[string]time.Time // source where we heard from the peer.

	// logging
	logger *log.Entry
}

func (p *Peer) String() string {
	return fmt.Sprintf("%s %s:%s", p.Hash, p.Address, p.ListenPort)
}

func (p *Peer) ConnectAddress() string {
	return fmt.Sprintf("%s:%s", p.Address, p.ListenPort)
}

func (p *Peer) StartToDial() {
	if !p.CanDial() {
		p.logger.Errorf("Maximum dial attempts reached")
		return
	}

	p.connectionAttempt = time.Now()
	p.connectionAttemptCount++

	if p.Location == 0 {
		loc, err := IP2Location(p.Address)
		if err != nil {
			p.logger.WithError(err).Warnf("Unable to convert address %s to location", p.Address)
			return
		}
		p.Location = loc
	}

	//p.stateMutex.Lock()
	//defer p.stateMutex.Unlock()

	p.connMutex.Lock()
	if p.conn != nil {
		p.logger.WithField("old_conn", p.conn).Warn("Peer started to dial despite not being offline")
		p.conn.Stop()
		p.conn = nil
	}
	p.connMutex.Unlock()

	p.IsOutgoing = true
	p.state = Connecting
	remote := fmt.Sprintf("%s:%s", p.Address, p.ListenPort)
	con, err := net.Dial("tcp", remote)
	p.logger.WithField("attempt", p.connectionAttemptCount).Debugf("Dialing to %s", remote)
	if err != nil {
		p.logger.WithError(err).Infof("Unable to connect to peer")
		return
	}

	p.startInternal(con)
}

func (p *Peer) StartWithActiveConnection(con net.Conn) {
	p.connMutex.Lock()
	if p.conn != nil {
		p.logger.WithField("old_conn", p.conn).Warn("Peer given new connection despite having old one")
		p.conn.Stop()
		p.conn = nil
	}
	p.connMutex.Unlock()
	//p.stateMutex.Lock()
	//defer p.stateMutex.Unlock()

	p.startInternal(con)
}

// startInternal is the common functionality for both dialing and accepting a connection
// is under locked mutex from superior function
func (p *Peer) startInternal(con net.Conn) {
	p.connMutex.Lock()
	p.conn = NewConnection(p.Hash, con, p.Receive, p.net)
	p.conn.Start()
	p.connMutex.Unlock()

	p.state = Online
	p.lastPeerRequest = time.Now()
	go p.monitorConnection() // this will die when the connection is closed
}

func (p *Peer) GoOffline() {
	//p.stateMutex.Lock()
	//defer p.stateMutex.Lock()
	p.state = Offline
	p.stop <- true
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

	// send this parcel from this peer
	parcel.Header.Network = p.net.conf.Network
	parcel.Header.Version = p.net.conf.ProtocolVersion
	parcel.Header.NodeID = p.config.NodeID
	parcel.Header.PeerPort = string(p.config.ListenPort) // notify other side of our port
	p.conn.Send.Send(parcel)
}

func (p *Peer) CanDial() bool {
	return p.connectionAttemptCount < p.config.RedialAttempts && p.ListenPort != ""
}

func (p *Peer) IsOnline() bool {
	//p.stateMutex.RLock()
	//defer p.stateMutex.RUnlock()
	return p.state == Online
}
func (p *Peer) IsOffline() bool {
	//p.stateMutex.RLock()
	//defer p.stateMutex.RUnlock()
	return p.state == Offline
}

// monitorConnection watches the underlying Connection.
// Any data arriving via connection will be passed on to the peer manager.
// If the connection dies, change state to Offline
func (p *Peer) monitorConnection() {
	defer func() {
		if r := recover(); r != nil {
			// p.conn ceased to exist
		}
	}()

Monitor:
	for {
		select {
		case <-p.stop: // manual stop, we need to tear down connection
			//p.stateMutex.Lock()
			//defer p.stateMutex.Lock()
			p.state = Offline
			p.connMutex.Lock()
			if p.conn != nil {
				p.conn.Stop()
				p.conn = nil
			}
			p.connMutex.Unlock()
			p.connectionAttemptCount = 0
			break Monitor
		case <-p.conn.Error: // if an error arrives here, the connection already stops itself
			p.state = Offline
			p.connMutex.Lock()
			if p.conn != nil {
				p.conn = nil
			}
			p.connMutex.Unlock()

			p.connectionAttemptCount = 0
			break Monitor
		case parcel := <-p.incoming:
			if newport := parcel.Header.PeerPort; newport != p.ListenPort {
				p.logger.WithFields(log.Fields{"old": p.ListenPort, "new": newport}).Debugf("Listen port changed")
				p.ListenPort = newport
			}
			if nodeid := parcel.Header.NodeID; nodeid != p.NodeID {
				p.logger.WithFields(log.Fields{"old": p.NodeID, "new": nodeid}).Debugf("NodeID changed")
				p.NodeID = nodeid
			}
			p.net.peerManager.Receive <- PeerParcel{Peer: p, Parcel: parcel} // TODO this is potentially blocking
		}
	}
}

// Better compares a peer to another peer to determine which one we
// would rather keep
//
// Prefers to keep peers that are online or connecting over peers that are not
// but if both are in the same state, it uses qualityscore
func (p *Peer) Better(other *Peer) bool {
	// TODO special
	/*	if p.IsSpecial() && !other.IsSpecial() {
		return true
	}*/

	if !p.IsOffline() && other.IsOffline() { // other is offline
		return true
	}

	if p.IsOnline() && !other.IsOnline() { // other is connecting
		return true
	}

	return p.QualityScore > other.QualityScore
}

func (p *Peer) AddressPort() string {
	return p.Address + ":" + p.Port
}

func (p *Peer) PeerIdent() string {
	return p.Hash[0:12] + "-" + p.Address + ":" + p.Port
}

func (p *Peer) PeerFixedIdent() string {
	address := fmt.Sprintf("%16s", p.Address)
	return p.Hash[0:12] + "-" + address + ":" + p.Port
}

func (p *Peer) PeerLogFields() log.Fields {
	return log.Fields{
		"address": p.Address,
		"port":    p.Port,
		//"peer_type": p.PeerTypeString(),
	}
}

// sort.Sort interface implementation
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

// sort.Sort interface implementation
type PeerDistanceSort []*Peer

func (p PeerDistanceSort) Len() int {
	return len(p)
}
func (p PeerDistanceSort) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p PeerDistanceSort) Less(i, j int) bool {
	return p[i].Location < p[j].Location
}
