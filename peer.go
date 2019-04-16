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

var peerLogger = packageLogger.WithField("subpack", "peer")

// Peer is a representation of another node in the network
//
// Peers exist in two forms: Temporary peers and regular peers
type Peer struct {
	net  *Network
	conn net.Conn
	//	endpoint  *Endpoint
	handshake Handshake

	// current state
	IsIncoming bool

	//	stateMutex             sync.RWMutex
	age         time.Time
	stop        chan bool
	stopSending chan bool

	lastPeerRequest time.Time
	lastPeerSend    time.Time
	receive         ParcelChannel
	send            ParcelChannel
	error           chan error
	disconnect      chan *Peer

	encoder *gob.Encoder // Wire format is gobs in this version, may switch to binary
	decoder *gob.Decoder // Wire format is gobs in this version, may switch to binary

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

	// logging
	logger *log.Entry
}

type PeerError struct {
	Peer *Peer
	err  error
}

func NewPeer(net *Network, addr string, hs Handshake, disconnect chan *Peer) *Peer {
	p := &Peer{}
	p.net = net
	p.disconnect = disconnect
	p.Address = addr
	p.handshake = hs

	p.Port = hs.Port
	p.NodeID = hs.NodeID
	p.Hash = fmt.Sprintf("%s:%s %x", p.Address, hs.Port, hs.NodeID)

	p.logger = peerLogger.WithFields(log.Fields{
		"node":    net.conf.NodeName,
		"hash":    p.Hash,
		"address": p.Address,
		"Port":    p.Port,
		"version": p.handshake.Version,
	})
	p.age = time.Now()
	p.stop = make(chan bool, 1)
	p.stopSending = make(chan bool, 1)
	p.receive = NewParcelChannel(net.conf.ChannelCapacity)
	p.send = NewParcelChannel(net.conf.ChannelCapacity)
	p.error = make(chan error, 10)
	//p.Hash = fmt.Sprintf("%x", net.rng.Int63())
	p.logger.Debugf("Creating new peer")
	return p
}

func (p *Peer) StartWithConnection(tcp net.Conn, incoming bool) {
	p.IsIncoming = incoming
	p.conn = tcp
	p.encoder = gob.NewEncoder(p.conn)
	p.decoder = gob.NewDecoder(p.conn)

	go p.monitorConnection()
	go p.readLoop()
	go p.sendLoop()
}

// Stop disconnects the peer from its active connection
func (p *Peer) Stop() {
	p.stop <- true
	p.stopSending <- true
}

/*func (p *Peer) dialInternal() {
	p.logger.Debugf("Dialing to %s:%s", p.Address, p.Port)

	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", p.net.conf.BindIP))
	if err != nil {
		p.logger.WithError(err).Errorf("Unable to resolve local interface \"%s:0\"", p.net.conf.BindIP)
		return
	}

	dialer := net.Dialer{
		LocalAddr: local,
		Timeout:   p.net.conf.DialTimeout,
	}
	con, err := dialer.Dial("tcp", fmt.Sprintf("%s:%s", p.Address, p.Port))

	if err != nil {
		p.logger.WithError(err).Infof("Unable to connect to peer")
		return
	}

	newcon := NewConnection(p.net, con, p.send, p.receive)
	newcon.Start()
	p.monitorConnection() // we're already in a goroutine
}*/

// monitorConnection watches the underlying Connection and the connection channel
//
// Any data arriving via connection will be passed on to the peer manager.
// If the connection dies, change state to Offline. If a new connection arrives, handle it.
func (p *Peer) monitorConnection() {
	for {
		select {
		case <-p.stop: // tear down the peer completely
			p.logger.Debug("Peer shutting down")
			if p.conn != nil {
				p.conn.Close()
			}
			p.conn = nil
			return
		case err := <-p.error:
			p.logger.WithError(err).Debug("Connection error")
			p.stop <- true
		case parcel := <-p.receive:
			p.QualityScore++
			p.logger.Debugf("Received incoming parcel: %v", parcel)
			p.LastReceive = time.Now()
			select {
			case p.net.peerParcel <- PeerParcel{Peer: p, Parcel: parcel}:
			default:
				p.logger.Warn("Peer manager unable to handle load, dropping")
			}
		}
	}
}

func (p *Peer) String() string {
	return p.Hash
}

func (p *Peer) Send(parcel *Parcel) {
	// send this parcel from this peer
	p.send.Send(parcel)
}

func (p *Peer) readLoop() {
	defer p.conn.Close() // close connection on fatal error
	for {
		var message Parcel

		p.conn.SetReadDeadline(time.Now().Add(p.net.conf.ReadDeadline))
		err := p.decoder.Decode(&message)
		if err != nil {
			p.error <- err
			return
		}

		p.LastReceive = time.Now()
		p.receive.Send(&message)
	}
}

// sendLoop listens to the Outgoing channel, pushing all data from there
// to the tcp connection
func (p *Peer) sendLoop() {
	defer p.conn.Close() // close connection on fatal error
	for {
		select {
		case <-p.stopSending:
			return
		case parcel := <-p.send:
			if parcel == nil {
				p.logger.Error("Received <nil> pointer")
				continue
			}

			p.conn.SetWriteDeadline(time.Now().Add(p.net.conf.WriteDeadline))
			err := p.encoder.Encode(parcel)
			if err != nil { // no error is recoverable
				p.error <- err
				return
			}
			p.LastSend = time.Now()
		}
	}
}
