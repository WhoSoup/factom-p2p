package p2p

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/whosoup/factom-p2p/util"
)

var peerLogger = packageLogger.WithField("subpack", "peer")

// Peer is an active connection to an endpoint in the network
type Peer struct {
	net  *Network
	conn net.Conn
	prot Protocol

	// current state
	IsIncoming bool

	stopper      sync.Once
	stop         chan bool
	stopDelivery chan bool

	lastPeerRequest time.Time
	peerShareAsk    bool
	lastPeerSend    time.Time
	send            ParcelChannel
	error           chan error
	status          chan peerStatus
	data            chan peerParcel

	connectionAttempt      time.Time
	connectionAttemptCount uint

	Seed bool

	IP     util.IP
	NodeID uint32 // a nonce to distinguish multiple nodes behind one IP address
	Hash   string // This is more of a connection ID than hash right now.

	// Metrics
	metricsMtx      sync.RWMutex
	Connected       time.Time
	QualityScore    int32     // 0 is neutral quality, negative is a bad peer.
	LastReceive     time.Time // Keep track of how long ago we talked to the peer.
	LastSend        time.Time // Keep track of how long ago we talked to the peer.
	ParcelsSent     uint64
	ParcelsReceived uint64
	BytesSent       uint64
	BytesReceived   uint64

	// logging
	logger *log.Entry
}

func NewPeer(net *Network, status chan peerStatus, data chan peerParcel) *Peer {
	p := &Peer{}
	p.net = net
	p.status = status
	p.data = data

	p.logger = peerLogger.WithFields(log.Fields{
		"node": net.conf.NodeName,
	})
	p.stop = make(chan bool, 1)
	p.stopDelivery = make(chan bool, 1)

	p.logger.Debugf("Creating blank peer")
	return p
}

func (p *Peer) bootstrapProtocol(hs *Handshake, conn net.Conn, decoder *gob.Decoder, encoder *gob.Encoder) error {
	v := hs.Header.Version
	if v > p.net.conf.ProtocolVersion {
		v = p.net.conf.ProtocolVersion
	}
	switch v {
	case 9:
		v9 := new(ProtocolV9)
		v9.init(p, conn, decoder, encoder)
		p.prot = v9

		// bootstrap
		p.lastPeerRequest = time.Time{}
		p.lastPeerSend = time.Time{}
		p.peerShareAsk = true

		hsParcel := new(Parcel)
		hsParcel.Address = hs.Header.TargetPeer
		hsParcel.Payload = hs.Payload
		hsParcel.Type = hs.Header.Type
		if !p.deliver(hsParcel) { // push the handshake to controller
			return fmt.Errorf("unable to deliver peer request to controller")
		}
	case 10:
		v10 := new(ProtocolV10)
		v10.init(p, conn, decoder, encoder)
		p.prot = v10
	default:
		return fmt.Errorf("unknown protocol version %d", v)
	}
	return nil
}

// StartWithHandshake performs a basic handshake maneouver to establish the validity
// of the connection. Immediately sends a Peer Request upon connection and waits for the
// response, which can be any parcel. The information in the header is verified, especially
// the port.
//
// The handshake ensures that ALL peers have a valid Port field to start with.
// If there is no reply within the specified HandshakeTimeout config setting, the process
// fails
func (p *Peer) StartWithHandshake(ip util.IP, con net.Conn, incoming bool) (bool, error) {
	tmplogger := p.logger.WithField("addr", ip.Address)
	timeout := time.Now().Add(p.net.conf.HandshakeTimeout)

	handshake := newHandshake(p.net.conf)

	decoder := gob.NewDecoder(con)
	encoder := gob.NewEncoder(con)
	con.SetWriteDeadline(timeout)
	con.SetReadDeadline(timeout)
	err := encoder.Encode(handshake)

	failfunc := func(err error) (bool, error) {
		tmplogger.WithError(err).Debug("Handshake failed")
		con.Close()
		return false, err
	}

	if err != nil {
		return failfunc(fmt.Errorf("Failed to send handshake to incoming connection"))
	}

	err = decoder.Decode(&handshake)
	if err != nil {
		return failfunc(fmt.Errorf("Failed to read handshake from incoming connection"))
	}

	// check basic structure
	if err = handshake.Valid(p.net.conf); err != nil {
		return failfunc(err)
	}

	if err = p.bootstrapProtocol(handshake, con, decoder, encoder); err != nil {
		return failfunc(err)
	}

	ip.Port = handshake.Header.PeerPort
	p.IP = ip
	p.NodeID = uint32(handshake.Header.NodeID)
	p.Hash = fmt.Sprintf("%s:%s %08x", ip.Address, ip.Port, p.NodeID)
	p.send = NewParcelChannel(p.net.conf.ChannelCapacity)
	p.error = make(chan error, 10)
	p.IsIncoming = incoming
	p.conn = con
	p.Connected = time.Now()
	p.logger = p.logger.WithFields(log.Fields{
		"hash":    p.Hash,
		"address": p.IP.Address,
		"Port":    p.IP.Port,
		"Version": p.prot.Version(),
	})

	p.status <- peerStatus{peer: p, online: true}

	go p.sendLoop()
	go p.readLoop()

	return true, nil
}

// Stop disconnects the peer from its active connection
func (p *Peer) Stop(andRemove bool) {
	p.stopper.Do(func() {
		p.logger.Debug("Stopping peer")
		sc := p.send

		p.send = nil
		p.error = nil

		p.stop <- true
		p.stopDelivery <- true

		if p.conn != nil {
			p.conn.Close()
		}

		if sc != nil {
			close(sc)
		}

		if andRemove {
			p.status <- peerStatus{peer: p, online: false}
		}
	})
}

func (p *Peer) String() string {
	return p.Hash
}

func (p *Peer) Send(parcel *Parcel) {
	p.send.Send(parcel)
}

func (p *Peer) quality(diff int32) int32 {
	p.metricsMtx.Lock()
	defer p.metricsMtx.Unlock()
	p.QualityScore += diff
	return p.QualityScore
}

func (p *Peer) readLoop() {
	if p.net.prom != nil {
		p.net.prom.ReceiveRoutines.Inc()
		defer p.net.prom.ReceiveRoutines.Dec()
	}
	defer p.conn.Close() // close connection on fatal error
	for {
		p.conn.SetReadDeadline(time.Now().Add(p.net.conf.ReadDeadline))
		msg, err := p.prot.Receive()
		if err != nil {
			p.logger.WithError(err).Debug("connection error (readLoop)")
			p.Stop(true)
			return
		}

		if err := msg.Valid(); err != nil {
			p.logger.WithError(err).Warnf("received invalid msg, disconnecting peer")
			p.Stop(true)
			if p.net.prom != nil {
				p.net.prom.Invalid.Inc()
			}
			return
		}

		p.metricsMtx.Lock()
		p.LastReceive = time.Now()
		p.QualityScore++
		p.ParcelsReceived++
		p.BytesReceived += uint64(len(msg.Payload))
		p.metricsMtx.Unlock()

		msg.Address = p.Hash
		if p.net.prom != nil {
			p.net.prom.ParcelsReceived.Inc()
			p.net.prom.ParcelSize.Observe(float64(len(msg.Payload)) / 1024)
			if msg.IsApplicationMessage() {
				p.net.prom.AppReceived.Inc()
			}
		}
		if !p.deliver(msg) {
			return
		}
	}
}

// deliver is a blocking delivery of this peer's messages to the peer manager.
func (p *Peer) deliver(parcel *Parcel) bool {
	select {
	case p.data <- peerParcel{peer: p, parcel: parcel}:
	case <-p.stopDelivery:
		return false
	}
	return true
}

// sendLoop listens to the Outgoing channel, pushing all data from there
// to the tcp connection
func (p *Peer) sendLoop() {
	if p.net.prom != nil {
		p.net.prom.SendRoutines.Inc()
		defer p.net.prom.SendRoutines.Dec()
	}

	defer p.conn.Close() // close connection on fatal error
	for {
		select {
		case <-p.stop:
			return
		case parcel := <-p.send:
			if parcel == nil {
				p.logger.Error("Received <nil> pointer")
				continue
			}

			p.conn.SetWriteDeadline(time.Now().Add(p.net.conf.WriteDeadline))
			err := p.prot.Send(parcel)
			if err != nil { // no error is recoverable
				p.logger.WithError(err).Debug("connection error (sendLoop)")
				p.Stop(true)
				return
			}

			p.metricsMtx.Lock()
			p.ParcelsSent++
			p.BytesSent += uint64(len(parcel.Payload))
			p.LastSend = time.Now()
			p.metricsMtx.Unlock()

			if p.net.prom != nil {
				p.net.prom.ParcelsSent.Inc()
				p.net.prom.ParcelSize.Observe(float64(len(parcel.Payload)+32) / 1024) // TODO FIX
				if parcel.IsApplicationMessage() {
					p.net.prom.AppSent.Inc()
				}
			}
		}
	}
}

func (p *Peer) GetMetrics() PeerMetrics {
	p.metricsMtx.RLock()
	defer p.metricsMtx.RUnlock()
	return PeerMetrics{
		Hash:             p.Hash,
		PeerQuality:      p.QualityScore,
		PeerAddress:      p.IP.Address,
		MomentConnected:  p.Connected,
		LastReceive:      p.LastReceive,
		LastSend:         p.LastSend,
		BytesReceived:    p.BytesReceived,
		BytesSent:        p.BytesSent,
		MessagesReceived: p.ParcelsReceived,
		MessagesSent:     p.ParcelsSent,
		Incoming:         p.IsIncoming,
	}
}
