package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/whosoup/factom-p2p/util"
)

// Network is the main access point for outside applications.
//
// ToNetwork is the channel over which to send parcels to the network layer
//
// FromNetwork is the channel that gets filled with parcels arriving from the network layer
type Network struct {
	ToNetwork   ParcelChannel
	FromNetwork ParcelChannel

	conf        *Configuration
	peerManager *peerManager

	stopRoute   chan bool
	peerParcel  chan PeerParcel
	listener    *util.LimitedListener
	metricsHook func(pm map[string]PeerMetrics)

	rng    *rand.Rand
	logger *log.Entry
}

var packageLogger = log.WithField("package", "p2p")

// DebugMessage is temporary
func (n *Network) DebugMessage() (string, string, int) {
	hv := ""
	r := fmt.Sprintf("\nONLINE:\n")
	s := n.peerManager.peers.Slice()
	count := len(s)
	for _, p := range s {

		r += fmt.Sprintf("\tPeer %s %d\n", p.String(), p.QualityScore)
		edge := ""
		if n.conf.NodeID < 4 || p.NodeID < 4 {
			min := n.conf.NodeID
			if p.NodeID < min {
				min = p.NodeID
			}
			if min != 0 {
				color := []string{"red", "green", "blue"}[min-1]
				edge = fmt.Sprintf(" {color:%s, weight=3}", color)
			}
		}
		if p.IsIncoming {
			hv += fmt.Sprintf("%s -> %s:%s%s\n", p.IP, n.conf.BindIP, n.conf.ListenPort, edge)
		} else {
			hv += fmt.Sprintf("%s:%s -> %s%s\n", n.conf.BindIP, n.conf.ListenPort, p.IP, edge)
		}
	}
	known := ""
	for _, ip := range n.peerManager.endpoints.IPs() {
		known += ip.Address + " "
	}
	r += "\nKNOWN:\n" + known
	return r, hv, count
}

// DebugServer is temporary
func DebugServer(n *Network) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		a, _, _ := n.DebugMessage()
		rw.Write([]byte(a))
	})

	mux.HandleFunc("/stats", func(rw http.ResponseWriter, req *http.Request) {
		out := ""
		out += fmt.Sprintf("Channels\n")
		out += fmt.Sprintf("\tToNetwork: %d / %d\n", len(n.ToNetwork), cap(n.ToNetwork))
		out += fmt.Sprintf("\tFromNetwork: %d / %d\n", len(n.FromNetwork), cap(n.FromNetwork))
		out += fmt.Sprintf("\tpeerParcel: %d / %d\n", len(n.peerParcel), cap(n.peerParcel))
		out += fmt.Sprintf("\nPeers (%d)\n", n.peerManager.peers.Total())
		for _, p := range n.peerManager.peers.Slice() {
			out += fmt.Sprintf("\t%s\n", p.IP)
			out += fmt.Sprintf("\t\tsend: %d / %d\n", len(p.send), cap(p.send))
			m := p.GetMetrics()
			out += fmt.Sprintf("\t\tBytesReceived: %d\n", m.BytesReceived)
			out += fmt.Sprintf("\t\tBytesSent: %d\n", m.BytesSent)
			out += fmt.Sprintf("\t\tMessagesSent: %d\n", m.MessagesSent)
			out += fmt.Sprintf("\t\tMessagesReceived: %d\n", m.MessagesReceived)
			out += fmt.Sprintf("\t\tMomentConnected: %s\n", m.MomentConnected)
			out += fmt.Sprintf("\t\tPeerQuality: %d\n", m.PeerQuality)
			out += fmt.Sprintf("\t\tIncoming: %v\n", m.Incoming)
			out += fmt.Sprintf("\t\tLastReceive: %s\n", m.LastReceive)
			out += fmt.Sprintf("\t\tLastSend: %s\n", m.LastSend)

		}

		out += fmt.Sprintf("\nEndpoints\n")
		for _, ep := range n.peerManager.endpoints.IPs() {
			out += fmt.Sprintf("\t%s", ep)
		}

		rw.Write([]byte(out))
	})

	go http.ListenAndServe("localhost:8070", mux)
}

// NewNetwork initializes a new network with the given configuration.
// The passed Configuration is copied and cannot be modified afterwards.
// Does not start the network automatically.
func NewNetwork(conf Configuration) *Network {
	myconf := conf // copy
	n := new(Network)
	n.logger = packageLogger.WithField("subpackage", "Network").WithField("node", conf.NodeName)

	n.conf = &myconf
	n.rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	if n.conf.NodeID == 0 {
		for n.conf.NodeID == 0 {
			n.conf.NodeID = n.rng.Uint64()
		}
		n.logger.Debugf("No node id specified, generated random node id %x")
	}

	n.peerManager = newPeerManager(n)

	n.peerParcel = make(chan PeerParcel, conf.ChannelCapacity)

	n.ToNetwork = NewParcelChannel(conf.ChannelCapacity)
	n.FromNetwork = NewParcelChannel(conf.ChannelCapacity)
	return n
}

// SetMetricsHook allows you to read peer metrics.
// Gets called approximately once a second and transfers the metrics
// of all CONNECTED peers in the format "identifying hash" -> p2p.PeerMetrics
func (n *Network) SetMetricsHook(f func(pm map[string]PeerMetrics)) {
	n.metricsHook = f
}

// Start starts the network.
// Listens to incoming connections on the specified port
// and connects to other peers
func (n *Network) Start() {
	n.logger.Info("Starting the P2P Network")
	n.peerManager.Start() // this will get peer manager ready to handle incoming connections
	n.stopRoute = make(chan bool, 1)
	DebugServer(n)
	go n.listenLoop()
	go n.routeLoop()
}

// Stop tears down all the active connections and stops the listener.
// Use before shutting down.
func (n *Network) Stop() {
	n.logger.Info("Stopping the P2P Network")
	n.peerManager.Stop()
	n.stopRoute <- true
	if n.listener != nil {
		n.listener.Close()
	}
}

// Merit rewards a peer for doing something right
func (n *Network) Merit(hash string) {
	n.logger.Debugf("Received merit for %s from application", hash)
	go n.peerManager.merit(hash)
}

// Demerit punishes a peer for doing something wrong. Too many demerits
// and the peer will be banned
func (n *Network) Demerit(hash string) {
	n.logger.Debugf("Received demerit for %s from application", hash)
	go n.peerManager.demerit(hash)
}

// Ban removes a peer as well as any other peer from that address
// and prevents any connection being established for the amount of time
// set in the configuration (default one week)
func (n *Network) Ban(hash string) {
	n.logger.Debugf("Received ban for %s from application", hash)
	go n.peerManager.ban(hash, n.conf.ManualBan)
}

// Disconnect severs connection for a specific peer. They are free to
// connect again afterward
func (n *Network) Disconnect(hash string) {
	n.logger.Debugf("Received disconnect for %s from application", hash)
	go n.peerManager.disconnect(hash)
}

// ParseSpecial takes a set of ip addresses that should be treated as special.
// Network will always attempt to have a connection to a special peer.
// Format is a single line of ip addresses separated by semicolon, eg
// "127.0.0.1;8.8.8.8;192.168.0.1"
func (n *Network) ParseSpecial(raw string) {
	n.logger.Debugf("Received new list of special peers from application: %s", raw)
	go n.peerManager.parseSpecial(raw)
}

// Total returns the number of active connections
func (n *Network) Total() int {
	return n.peerManager.peers.Total()
}

// routeLoop Takes messages from the network's ToNetwork channel and routes it
// to the peerManager via the appropriate function
func (n *Network) routeLoop() {
	for {
		// TODO metrics?
		// blocking read on ToNetwork, and c.stopRoute
		select {
		case message := <-n.ToNetwork:
			switch message.Header.TargetPeer {
			case FullBroadcastFlag:
				n.peerManager.Broadcast(message, true)
			case BroadcastFlag:
				n.peerManager.Broadcast(message, false)
			case RandomPeerFlag:
				n.peerManager.ToPeer("", message)
			default:
				n.peerManager.ToPeer(message.Header.TargetPeer, message)
			}
		// stop this loop if anything shows up
		case <-n.stopRoute:
			return
		}
	}
}

// listenLoop listens for incoming TCP connections and passes them off to peer manager
func (n *Network) listenLoop() {
	tmpLogger := n.logger.WithFields(log.Fields{"address": n.conf.BindIP, "port": n.conf.ListenPort})
	tmpLogger.Debug("controller.listenLoop() starting up")

	addr := fmt.Sprintf("%s:%s", n.conf.BindIP, n.conf.ListenPort)

	l, err := util.NewLimitedListener(addr, n.conf.ListenLimit)
	if err != nil {
		tmpLogger.WithError(err).Error("controller.Start() unable to start limited listener")
		return
	}

	n.listener = l

	// start permanent loop
	// terminates on program exit or when listener is closed
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && !ne.Timeout() {
				if !ne.Temporary() {
					tmpLogger.WithError(err).Warn("controller.acceptLoop() error accepting")
					return
				}
			}
			continue
		}

		go n.peerManager.HandleIncoming(conn)
	}
}
