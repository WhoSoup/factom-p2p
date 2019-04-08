package p2p

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Network struct {
	running      bool
	runningMutex sync.Mutex

	conf        *Configuration
	controller  *controller
	peerManager *peerManager

	ToNetwork   ParcelChannel
	FromNetwork ParcelChannel

	logger *log.Entry
}

var packageLogger = log.WithField("package", "p2p")

func (n *Network) DebugMessage() (string, string) {
	hv := ""
	r := ""
	for _, p := range n.peerManager.peers.Slice() {

		if p.IsOffline() {
			continue
		}

		r += fmt.Sprintf("\tPeer %s %v\n", p.String(), p.state.String())
		//		r += fmt.Sprintf("\t\tLast Send: %s\n", p.LastSend)
		//		r += fmt.Sprintf("\t\tLast Revc: %s\n", p.LastReceive)
		if p.IsOutgoing {
			hv += fmt.Sprintf("%s -> %s\n", n.conf.BindIP, p.Address)
		}
	}
	return r, hv
}

func NewNetwork(conf Configuration) *Network {
	myconf := conf
	n := new(Network)
	n.logger = packageLogger.WithField("subpackage", "Network")

	n.conf = &myconf
	n.controller = newController(n)
	n.peerManager = newPeerManager(n)

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

		go n.peerManager.Start() // this will get peer manager ready to handle incoming connections
		go n.controller.Start()
	}
}

func (n *Network) Stop() {
	n.runningMutex.Lock()
	defer n.runningMutex.Unlock()
	n.peerManager.Stop()
	n.controller.Stop()
	n.running = false
}
