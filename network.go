package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type Network struct {
	ToNetwork   ParcelChannel
	FromNetwork ParcelChannel

	conf        *Configuration
	peerManager *peerManager

	stopRoute   chan bool
	peerParcel  chan PeerParcel
	listener    *LimitedListener
	location    uint32
	metricsHook func(pm map[string]PeerMetrics)

	rng    *rand.Rand
	logger *log.Entry
}

var packageLogger = log.WithField("package", "p2p")

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
		out += "\nPeers\n"
		for _, p := range n.peerManager.peers.Slice() {
			out += fmt.Sprintf("\t%s\n", p.IP)
			out += fmt.Sprintf("\t\tsend: %d / %d\n", len(p.send), cap(p.send))
		}

		rw.Write([]byte(out))
	})

	go http.ListenAndServe("localhost:8070", mux)
}

func NewNetwork(conf Configuration) *Network {
	myconf := conf // copy
	n := new(Network)
	n.logger = packageLogger.WithField("subpackage", "Network").WithField("node", conf.NodeName)

	n.conf = &myconf

	n.peerManager = newPeerManager(n)
	n.rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	if n.conf.BindIP != "" {
		n.location, _ = IP2Location(n.conf.BindIP)
	}

	n.peerParcel = make(chan PeerParcel, conf.ChannelCapacity)

	n.ToNetwork = NewParcelChannel(conf.ChannelCapacity)
	n.FromNetwork = NewParcelChannel(conf.ChannelCapacity)
	return n
}

func (n *Network) SetMetricsHook(f func(pm map[string]PeerMetrics)) {
	n.metricsHook = f
}

// Start initializes the network by starting the peer manager and listening to incoming connections
func (n *Network) Start() {
	n.logger.Info("Starting the P2P Network")
	n.peerManager.Start() // this will get peer manager ready to handle incoming connections
	n.stopRoute = make(chan bool, 1)
	DebugServer(n)
	go n.listenLoop()
	go n.routeLoop()
}

func (n *Network) Stop() {
	n.logger.Info("Stopping the P2P Network")
	n.peerManager.Stop()
	n.stopRoute <- true
	if n.listener != nil {
		n.listener.Close()
	}
}

func (n *Network) Merit(hash string) {
	n.logger.Debugf("Received merit for %s from application", hash)
	go n.peerManager.merit(hash)
}

func (n *Network) Demerit(hash string) {
	n.logger.Debugf("Received demerit for %s from application", hash)
	go n.peerManager.demerit(hash)
}

func (n *Network) Ban(hash string) {
	n.logger.Debugf("Received ban for %s from application", hash)
	go n.peerManager.ban(hash, n.conf.ManualBan)
}

func (n *Network) Disconnect(hash string) {
	n.logger.Debugf("Received disconnect for %s from application", hash)
	go n.peerManager.disconnect(hash)
}

func (n *Network) ParseSpecial(raw string) {
	n.logger.Debugf("Received new list of special peers from application: %s", raw)
	go n.peerManager.parseSpecial(raw)
}

func (n *Network) Total() int {
	return n.peerManager.peers.Total()
}

// routeLoop Takes messages from the network's ToNetwork channel and routes it
// to the peerManager via the appropriate function
func (n *Network) routeLoop() {
	for {
		// TODO metrics?
		// blocking read on ToNetwork, and c.stop
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

	l, err := NewLimitedListener(addr, n.conf.ListenLimit)
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
					tmpLogger.WithError(err).Warn("controller.acceptLoop() error accepting, shutting down")
					return
				}
			}
			continue
		}

		// currently a non-concurrent implementation
		n.peerManager.HandleIncoming(conn)
	}
}
