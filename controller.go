package p2p

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var controllerLogger = packageLogger.WithField("subpack", "controller")

// controller is responsible for managing Peers and Endpoints
type controller struct {
	net *Network

	peerStatus chan peerStatus
	peerData   chan peerParcel

	peers    *PeerStore
	dialer   *Dialer
	listener *LimitedListener

	specialMtx   sync.RWMutex
	specialCount int

	banMtx  sync.RWMutex
	Bans    map[string]time.Time // (ip|ip:port) => time the ban ends
	Special map[string]bool      // (ip|ip:port) => bool

	shareListener map[*Peer]func(*Parcel)
	shareMtx      sync.RWMutex

	lastPeerDial time.Time
	lastPersist  time.Time

	counterMtx sync.RWMutex
	online     int
	connecting int

	lastRound    time.Time
	seed         *seed
	replenishing bool
	rounds       int // TODO make prometheus

	logger *log.Entry
}

// newController creates a new controller
// configuration is shared between the two
func newController(network *Network) (*controller, error) {
	var err error
	c := &controller{}
	c.net = network
	conf := network.conf

	c.logger = controllerLogger.WithFields(log.Fields{
		"node":    conf.NodeName,
		"port":    conf.ListenPort,
		"network": conf.Network})
	c.logger.Debugf("Initializing Controller")

	c.dialer, err = NewDialer(conf.BindIP, conf.RedialInterval, conf.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dialer: %v", err)
	}
	c.lastPersist = time.Now()

	c.peerStatus = make(chan peerStatus, 10) // TODO reconsider this value
	c.peerData = make(chan peerParcel, c.net.conf.ChannelCapacity)

	c.Bans = make(map[string]time.Time)
	c.Special = make(map[string]bool)
	c.shareListener = make(map[*Peer]func(*Parcel))

	// CAT
	c.lastRound = time.Now()
	c.seed = newSeed(c.net.conf.SeedURL, c.net.conf.PeerReseedInterval)

	c.bootStrapPeers()
	c.addSpecial(c.net.conf.Special)

	if c.net.prom != nil {
		c.net.prom.KnownPeers.Set(float64(c.peers.Total()))
	}

	return c, nil
}

// ban bans the peer indicated by the hash as well as any other peer from that ip
// address
func (c *controller) ban(hash string, duration time.Duration) {
	peer := c.peers.Get(hash)
	if peer != nil {
		c.banMtx.Lock()

		end := time.Now().Add(duration)

		// there's a stronger ban in place already
		if existing, ok := c.Bans[peer.Endpoint.IP]; ok && end.Before(existing) {
			end = existing
		}

		c.Bans[peer.Endpoint.IP] = end
		c.Bans[peer.Endpoint.String()] = end

		for _, p := range c.peers.Slice() {
			if p.Endpoint.IP == peer.Endpoint.IP {
				peer.Stop()
			}
		}
		c.banMtx.Unlock()
	}
}

func (c *controller) banEndpoint(ep Endpoint, duration time.Duration) {
	c.banMtx.Lock()
	c.Bans[ep.String()] = time.Now().Add(duration)
	c.banMtx.Unlock()

	if duration > 0 {
		for _, p := range c.peers.Slice() {
			if p.Endpoint == ep {
				p.Stop()
			}
		}
	}
}

func (c *controller) isBannedEndpoint(ep Endpoint) bool {
	c.banMtx.RLock()
	defer c.banMtx.RUnlock()
	return time.Now().Before(c.Bans[ep.IP]) || time.Now().Before(c.Bans[ep.String()])
}

func (c *controller) isBannedIP(ip string) bool {
	c.banMtx.RLock()
	defer c.banMtx.RUnlock()
	return time.Now().Before(c.Bans[ip])
}

func (c *controller) isSpecial(ep Endpoint) bool {
	c.specialMtx.RLock()
	defer c.specialMtx.RUnlock()
	return c.Special[ep.String()]
}
func (c *controller) isSpecialIP(ip string) bool {
	c.specialMtx.RLock()
	defer c.specialMtx.RUnlock()
	return c.Special[ip]
}

func (c *controller) disconnect(hash string) {
	peer := c.peers.Get(hash)
	if peer != nil {
		peer.Stop()
	}
}

func (c *controller) addSpecial(raw string) {
	if len(raw) == 0 {
		return
	}
	specialEndpoints := c.parseSpecial(raw)
	c.specialMtx.Lock()
	for _, ep := range specialEndpoints {
		c.logger.Debugf("Registering special endpoint %s", ep)
		c.Special[ep.String()] = true
		c.Special[ep.IP] = true
	}
	c.specialCount = len(c.Special)
	c.specialMtx.Unlock()
}

func (c *controller) parseSpecial(raw string) []Endpoint {
	var eps []Endpoint
	split := strings.Split(raw, ",")
	for _, item := range split {
		ep, err := ParseEndpoint(item)
		if err != nil {
			c.logger.Warnf("unable to determine host and port of special entry \"%s\"", item)
			continue
		}
		eps = append(eps, ep)
	}
	return eps
}

// Start starts the controller
// reads from the seed and connect to peers
func (c *controller) Start() {
	c.logger.Info("Starting the Controller")

	go c.run()
	go c.manageData()
	go c.manageOnline()
	go c.listen()
	go c.catReplenish()
}

func (c *controller) bootStrapPeers() {
	c.peers = NewPeerStore()
}

func (c *controller) manageData() {
	c.logger.Debug("Start manageData()")
	defer c.logger.Debug("Stop manageData()")
	for {
		select {
		case pp := <-c.peerData:
			parcel := pp.parcel
			peer := pp.peer

			if peer == nil && !parcel.IsApplicationMessage() { // peer disconnected between sending message and now
				c.logger.Debugf("Received parcel %s from peer not in system", parcel)
				continue
			}

			//c.logger.Debugf("Received parcel %s from %s", parcel, peer)
			switch parcel.Type {
			case TypePing:
				go func() {
					parcel := newParcel(TypePong, []byte("Pong"))
					peer.Send(parcel)
				}()
			case TypeMessage:
				//c.net.FromNetwork.Send(parcel)
				fallthrough
			case TypeMessagePart:
				parcel.Type = TypeMessage
				c.net.FromNetwork.Send(parcel)
			case TypePeerRequest:
				if time.Since(peer.lastPeerSend) >= c.net.conf.PeerRequestInterval {
					peer.lastPeerSend = time.Now()
					go c.sharePeers(peer)
				} else {
					c.logger.Warnf("peer %s sent a peer request too early", peer)
				}
			case TypePeerResponse:
				c.shareMtx.RLock()
				if f, ok := c.shareListener[peer]; ok {
					f(parcel)
				}
				c.shareMtx.RUnlock()
			default:
				//not handled
			}
		}
	}
}

// Broadcast delivers a parcel to multiple connections specified by the fanout.
// A full broadcast sends the parcel to ALL connected peers
func (c *controller) Broadcast(parcel *Parcel, full bool) {
	if full {
		for _, p := range c.peers.Slice() {
			p.Send(parcel)
		}
		return
	}
	selection := c.selectBroadcastPeers(c.net.conf.Fanout)
	for _, p := range selection {
		p.Send(parcel)
	}
}

// ToPeer sends a parcel to a single peer, specified by their peer hash.
// If the hash is empty, a random connected peer will be chosen
func (c *controller) ToPeer(hash string, parcel *Parcel) {
	if hash == "" {
		if random := c.randomPeer(); random != nil {
			random.Send(parcel)
		} else {
			c.logger.Warnf("attempted to send parcel %s to a random peer but no peers are connected", parcel)
		}
	} else {
		p := c.peers.Get(hash)
		if p != nil {
			p.Send(parcel)
		}
	}
}

func (c *controller) randomPeers(count uint) []*Peer {
	peers := c.peers.Slice()
	// not enough to randomize
	if uint(len(peers)) <= count {
		return peers
	}

	c.net.rng.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	return peers[:count]
}

func (c *controller) randomPeer() *Peer {
	peers := c.randomPeers(1)
	if len(peers) == 1 {
		return peers[0]
	}
	return nil
}
