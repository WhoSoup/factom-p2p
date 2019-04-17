package p2p

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Network struct {
	ToNetwork   ParcelChannel
	FromNetwork ParcelChannel

	running      bool
	runningMutex sync.Mutex

	conf        *Configuration
	controller  *controller
	peerManager *peerManager

	location uint32

	peerParcel chan PeerParcel
	peerStatus chan PeerStatus

	rng *rand.Rand

	logger *log.Entry
}

var packageLogger = log.WithField("package", "p2p")

func (n *Network) DebugMessage() (string, string) {
	hv := ""
	r := ""

	/*	for _, p := range n.peerManager.tempPeers.Slice() {
		r += fmt.Sprintf("\tPeer %s %v %v %d\n", p.String(), p.state.String(), p.Temporary, p.QualityScore)
	}*/
	r += "\nONLINE:\n"
	for _, p := range n.peerManager.peers.Slice() {
		r += fmt.Sprintf("\tPeer %s %v %d\n", p.String(), p.Temporary, p.QualityScore)
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
			hv += fmt.Sprintf("%s -> %s%s\n", p.Address, n.conf.BindIP, edge)
		} else {
			hv += fmt.Sprintf("%s -> %s%s\n", n.conf.BindIP, p.Address, edge)
		}
	}
	known := ""
	for ip := range n.peerManager.endpoints.known {
		known += ip + " "
	}
	r += "\nKNOWN:\n" + known
	return r, hv
}

func NewNetwork(conf Configuration) *Network {
	myconf := conf
	n := new(Network)
	n.logger = packageLogger.WithField("subpackage", "Network")

	n.conf = &myconf

	n.controller = newController(n)
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

// Start initializes the network by starting the peer manager and listening to incoming connections
func (n *Network) Start() {
	n.runningMutex.Lock()
	defer n.runningMutex.Unlock()
	if n.running {
		n.logger.Error("Tried to start the P2P Network even though it's already running")
	} else {
		n.running = true
		n.logger.Info("Starting the P2P Network")

		n.peerManager.Start() // this will get peer manager ready to handle incoming connections
		n.controller.Start()
	}
}

func (n *Network) Stop() {
	n.runningMutex.Lock()
	defer n.runningMutex.Unlock()
	n.peerManager.Stop()
	n.controller.Stop()
	n.running = false
}
