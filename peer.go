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
	p := &Peer{Address: address, state: Offline}
	p.net = net
	p.logger = peerLogger.WithFields(log.Fields{
		"node":    net.conf.NodeName,
		"hash":    p.Hash,
		"address": p.Address,
		"Port":    p.Port,
	})
	p.age = time.Now()
	p.stop = make(chan interface{}, 1)
	p.Receive = NewParcelChannel(net.conf.ChannelCapacity)
	p.Hash = fmt.Sprintf("%x", net.rng.Int63())
	p.logger.Debugf("Creating new peer")
	return p
}

func (p *Peer) ImportMetrics(other *Peer) {
	p.QualityScore = other.QualityScore
	if other.Seed {
		p.Seed = other.Seed
	}
	p.age = other.age
	p.Hash = other.Hash
	if other.Dialable {
		p.Dialable = true
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

func (p *Peer) canUpgrade() bool {
	return p.Temporary && p.Port != "0" && p.Port != ""
}

func (p *Peer) StartToDial() {
	if !p.CanDial() {
		p.logger.Errorf("Maximum dial attempts reached")
		return
	}
	if p.state == Connecting {
		p.logger.Debug("Still attempting to connect")
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

	p.connMutex.Lock()
	if p.conn != nil {
		p.logger.WithField("old_conn", p.conn).Warn("Peer started to dial despite not being offline")
		p.conn.Stop()
		p.conn = nil
	}
	p.connMutex.Unlock()

	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", p.net.conf.BindIP))
	if err != nil {
		p.logger.WithError(err).Errorf("Unable to resolve local interface \"%s:0\"", p.net.conf.BindIP)
	}

	p.IsOutgoing = true
	p.state = Connecting
	p.logger.Debugf("State changed to %s", p.state.String())
	remote, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", p.Address, p.Port))
	if err != nil {
		p.logger.WithError(err).Infof("Unable to resolve remote address \"%s:%s\"", p.Address, p.Port)
		return
	}
	con, err := net.DialTCP("tcp", local, remote)
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
	p.logger.Debugf("State changed to %s", p.state.String())
	p.lastPeerRequest = time.Now()
	go p.monitorConnection() // this will die when the connection is closed
}

func (p *Peer) GoOffline() {
	if p.state == Offline {
		return
	}
	p.state = Offline
	p.logger.Debugf("State changed to %s", p.state.String())
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
			p.logger.Debug("Manual stop")
			p.state = Offline
			p.connMutex.Lock()
			if p.conn != nil {
				p.conn.Stop()
				p.conn = nil
			}
			p.connMutex.Unlock()
			p.connectionAttemptCount = 0
			break Monitor
		case err := <-p.conn.Error: // if an error arrives here, the connection already stops itself
			p.logger.WithError(err).Debug("Connection error")
			p.state = Offline
			p.logger.Debugf("State changed to %s", p.state.String())
			p.connMutex.Lock()
			if p.conn != nil {
				p.conn = nil
			}
			p.connMutex.Unlock()

			p.connectionAttemptCount = 0
			break Monitor
		case parcel := <-p.Receive:
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
			p.net.peerManager.Receive <- PeerParcel{Peer: p, Parcel: parcel} // TODO this is potentially blocking
		}
	}
}

func (p *Peer) Shareable() bool {
	return p.QualityScore >= p.net.conf.MinimumQualityScore && !p.Temporary && !p.Seed
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
