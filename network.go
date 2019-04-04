package p2p

import (
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

func NewNetwork(conf Configuration) *Network {
	myconf := conf
	n := new(Network)
	n.logger = packageLogger.WithField("subpackage", "Network")

	n.conf = &myconf
	n.controller = newController(n)
	n.peerManager = newpeerManager(n)

	n.ToNetwork = NewParcelChannel(conf.ChannelCapacity)
	return n
}

// Start initializes the network by starting the peer manager and listening to incoming connections
func (n *Network) Start() {
	n.runningMutex.Lock()
	defer n.runningMutex.Unlock()
	n.logger.Info("Starting the P2P Network")

	go n.peerManager.Start() // this will get peer manager ready to handle incoming connections
	go n.controller.Start()

}
