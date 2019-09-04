package p2p

import (
	"bufio"
	"encoding/json"
	"os"
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

	stopPeers  chan bool
	stopData   chan bool
	stopOnline chan bool
	stopFill   chan bool
	dial       chan IP

	peers     *PeerStore
	endpoints *Endpoints
	dialer    *Dialer
	listener  *LimitedListener

	specialMtx   sync.RWMutex
	specialCount int

	lastPeerDial    time.Time
	lastSeedRefresh time.Time
	lastPersist     time.Time

	counterMtx sync.RWMutex
	online     int
	connecting int

	cat          *cat
	lastRound    time.Time
	seed         *seed
	replenishing bool
	rounds       int // TODO make prometheus

	logger *log.Entry
}

// newController creates a new controller
// configuration is shared between the two
func newController(network *Network) *controller {
	c := &controller{}
	c.net = network
	conf := network.conf

	c.logger = controllerLogger.WithFields(log.Fields{
		"node":    conf.NodeName,
		"port":    conf.ListenPort,
		"network": conf.Network})
	c.logger.Debugf("Initializing Controller")

	c.dialer = NewDialer(conf.BindIP, conf.RedialInterval, conf.RedialReset, conf.DialTimeout, conf.RedialAttempts)
	c.lastPersist = time.Now()

	c.peerStatus = make(chan peerStatus, 10) // TODO reconsider this value
	c.peerData = make(chan peerParcel, c.net.conf.ChannelCapacity)

	c.stopPeers = make(chan bool, 1)
	c.stopData = make(chan bool, 1)
	c.stopOnline = make(chan bool, 1)
	c.stopFill = make(chan bool, 1)
	//c.dial = make(chan IP, 50)

	// CAT
	c.cat = newCat(c.net)
	c.lastRound = time.Now()
	c.seed = newSeed(c.net.conf.SeedURL)

	c.bootStrapPeers()
	c.addSpecial(c.net.conf.Special)

	if c.net.prom != nil {
		c.net.prom.KnownPeers.Set(float64(c.endpoints.Total()))
	}

	return c
}

// ban bans the peer indicated by the hash as well as any other peer from that ip
// address
func (c *controller) ban(hash string, duration time.Duration) {
	peer := c.peers.Get(hash)
	if peer != nil {
		c.endpoints.BanAddress(peer.IP.Address, time.Now().Add(duration))
		for _, p := range c.peers.Slice() {
			if p.IP.Address == peer.IP.Address {
				peer.Stop(true)
			}
		}
	}
}

func (c *controller) merit(hash string) {
	peer := c.peers.Get(hash)
	if peer != nil {
		peer.quality(1)
	}
}

func (c *controller) demerit(hash string) {
	peer := c.peers.Get(hash)
	if peer != nil {
		if peer.quality(-1) < c.net.conf.MinimumQualityScore {
			c.ban(hash, c.net.conf.AutoBan)
		}
	}
}

func (c *controller) disconnect(hash string) {
	peer := c.peers.Get(hash)
	if peer != nil {
		peer.Stop(true)
	}
}

func (c *controller) addSpecial(raw string) {
	if len(raw) == 0 {
		return
	}
	adds := c.parseSpecial(raw)
	for _, add := range adds {
		c.logger.Debugf("Registering special endpoint %s", add)
		c.endpoints.Register(add, "Special")
	}

	c.specialMtx.Lock()
	c.specialCount += len(adds)
	c.specialMtx.Unlock()
}

func (c *controller) parseSpecial(raw string) []IP {
	var ips []IP
	split := strings.Split(raw, ",")
	for _, item := range split {
		ip, err := ParseAddress(item)
		if err != nil {
			c.logger.Warnf("unable to determine host and port of special entry \"%s\"", item)
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

// Start starts the controller
// reads from the seed and connect to peers
func (c *controller) Start() {
	c.logger.Info("Starting the Controller")

	go c.managePeers()
	go c.manageData()
	go c.manageOnline()
	go c.listen()
	//go c.fillLoop()
}

// Stop shuts down the controller and all active connections
func (c *controller) Stop() {
	c.stopData <- true
	c.stopPeers <- true
	c.stopOnline <- true
	c.stopFill <- true

	if c.listener != nil {
		c.listener.Close()
	}

	for _, p := range c.peers.Slice() {
		p.Stop(false)
		c.peers.Remove(p)
	}
}

func (c *controller) bootStrapPeers() {
	c.peers = NewPeerStore()
	c.endpoints = c.loadEndpoints() // creates blank if none exist
	c.lastSeedRefresh = time.Now()
	c.reseed()
}

func (c *controller) manageData() {
	c.logger.Debug("Start manageData()")
	defer c.logger.Debug("Stop manageData()")
	for {
		select {
		case <-c.stopData:
			return
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
					c.logger.Warnf("Peer %s is hammering peer requests", peer)
				}
			case TypePeerResponse:
				if peer.peerShareDeliver != nil { // they have a special channel, aka we asked!!
					select {
					case peer.peerShareDeliver <- parcel: // nonblocking
					default:
					}
					peer.peerShareDeliver = nil
					//go c.processPeers(peer, parcel)
				} else {
					c.logger.Warnf("Peer %s sent us an umprompted peer share", peer)
				}
			default:
				//not handled
			}
		}
	}
}

// processPeers processes a peer share response
func (c *controller) processPeers(peer *Peer, parcel *Parcel) []IP {
	list, err := peer.prot.ParsePeerShare(parcel.Payload)

	if err != nil {
		c.logger.WithError(err).Warnf("Failed to unmarshal peer share from peer %s", peer)
	}

	c.logger.Debugf("Received peer share from %s: %+v", peer, list)

	// cycles through list twice but we don't want to add any if one of them is bad
	for _, p := range list {
		if !p.Verify() {
			c.logger.Infof("Peer %s tried to send us peer share with bad data: %s", peer, p)
			return nil
		}
	}

	var res []IP
	for _, p := range list {
		ip, err := NewIP(p.Address, p.Port)
		if err != nil {
			c.logger.WithError(err).Infof("Unable to register endpoint %s:%s from peer %s", p.Address, p.Port, peer)
		} else if !c.endpoints.BannedEndpoint(ip) {
			//c.endpoints.Register(ip, peer.IP.Address)
			res = append(res, ip)
		}
	}

	if c.net.prom != nil {
		c.net.prom.KnownPeers.Set(float64(c.endpoints.Total()))
	}

	return res
}

// sharePeers creates a list of peers to share and sends it to peer
func (c *controller) sharePeers(peer *Peer) {
	if peer == nil {
		return
	}
	// CAT select n random active peers
	var list []IP
	tmp := c.peers.Slice()
	for _, i := range c.net.rng.Perm(len(tmp)) {
		if tmp[i] == nil { // TODO investigate why this happens
			continue
		}
		if tmp[i].Hash == peer.Hash {
			continue
		}
		list = append(list, tmp[i].IP)
		if uint(len(tmp)) >= c.net.conf.PeerShareAmount {
			break
		}
	}

	payload, err := peer.prot.MakePeerShare(list)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal peer list to json")
		return
	}
	c.logger.Debugf("Sharing %d peers with %s", len(list), peer)
	parcel := newParcel(TypePeerResponse, payload)
	peer.Send(parcel)
}

func (c *controller) reseed() {
	if uint(c.peers.Total()) < c.net.conf.MinReseed {
		seeds := c.seed.retrieve()
		for _, ip := range seeds {
			go c.Dial(ip)
		}
	}
}

func (c *controller) catRound() {
	c.logger.Debug("Cat Round")
	c.reseed()

	c.rounds++

	peers := c.peers.Slice()
	toDrop := len(peers) - int(c.net.conf.Drop)

	if toDrop > 0 {
		perm := c.net.rng.Perm(len(peers))

		dropped := 0
		for _, i := range perm {
			if c.endpoints.IsSpecial(peers[i].IP) {
				continue
			}
			peers[i].Stop(true)
			dropped++
			if dropped >= toDrop {
				break
			}
		}
	}

	go c.catReplenish()

}

func (c *controller) catReplenish() {
	if c.replenishing {
		return
	}
	c.replenishing = true
	for uint(c.peers.Total()) < c.net.conf.Target {
		p := c.selectRandomPeer()

		if p == nil { // no peers connected
			time.Sleep(time.Second)
			continue
		}

		async := make(chan *Parcel, 1)
		p.peerShareDeliver = async

		req := newParcel(TypePeerRequest, []byte("Peer Request"))
		p.Send(req)

		select {
		case resp := <-async:
			ips := c.processPeers(p, resp)
			for _, ip := range ips {
				c.Dial(ip) // NOT A GOROUTINE, wait for it to finish
			}
		case <-time.After(time.Second * 10):
		}
		p.peerShareDeliver = nil
	}
	c.replenishing = false
}

// managePeers is responsible for everything that involves proactive management
// not based on reactions. runs once a second
func (c *controller) managePeers() {
	c.logger.Debug("Start managePeers()")
	defer c.logger.Debug("Stop managePeers()")

	for {
		if time.Since(c.lastPersist) > c.net.conf.PersistInterval {
			c.lastPersist = time.Now()

			// TODO persist peers instead
			//err := c.endpoints.Persist(c.net.conf.PersistFile, c.net.conf.PersistLevel, c.net.conf.PersistMinimum, c.net.conf.PersistAgeLimit)
			//if err != nil {
			//	c.logger.WithError(err).Errorf("unable to persist peers")
			//}
		}

		// CAT rounds
		if time.Since(c.lastRound) > c.net.conf.RoundTime {
			c.lastRound = time.Now()

			c.catRound()
		}

		metrics := make(map[string]PeerMetrics)
		for _, p := range c.peers.Slice() {
			metrics[p.Hash] = p.GetMetrics()

			if time.Since(p.LastSend) > c.net.conf.PingInterval {
				ping := newParcel(TypePing, []byte("Ping"))
				p.Send(ping)
			}
		}

		if c.net.metricsHook != nil {
			go c.net.metricsHook(metrics)
		}

		select {
		case <-c.stopPeers:
			return
		case <-time.After(time.Second):
		}
	}
}

func (c *controller) selectRandomPeers(count uint) []*Peer {
	peers := c.peers.Slice()

	// not enough to randomize
	if uint(len(peers)) <= count {
		return peers
	}

	var special []*Peer
	var regular []*Peer

	for _, p := range peers {
		if c.endpoints.IsSpecial(p.IP) {
			special = append(special, p)
		} else {
			regular = append(regular, p)
		}
	}

	if uint(len(regular)) < count {
		return append(special, regular...)
	}

	c.net.rng.Shuffle(len(regular), func(i, j int) {
		regular[i], regular[j] = regular[j], regular[i]
	})

	return append(special, peers[:count]...)
}

func (c *controller) selectRandomPeer() *Peer {
	peers := c.peers.Slice()
	if len(peers) == 0 {
		return nil
	}
	if len(peers) == 1 {
		return peers[0]
	}

	return peers[c.net.rng.Intn(len(peers))]
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
	selection := c.selectRandomPeers(c.net.conf.Fanout)
	for _, p := range selection {
		p.Send(parcel)
	}
}

// ToPeer sends a parcel to a single peer, specified by their peer hash.
// If the hash is empty, a random connected peer will be chosen
func (c *controller) ToPeer(hash string, parcel *Parcel) {
	if hash == "" {
		if random := c.selectRandomPeer(); random != nil {
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

func (c *controller) loadEndpoints() *Endpoints {
	eps := NewEndpoints()

	path := c.net.conf.PersistFile
	if path == "" {
		return eps
	}
	c.logger.Debugf("Attempting to parse file %s for endpoints", path)

	file, err := os.Open(path)
	if err != nil {
		c.logger.WithError(err).Errorf("loadEndpoints(): file open error for %s", path)
		return eps
	}

	dec := json.NewDecoder(bufio.NewReader(file))
	err = dec.Decode(eps)

	if err != nil {
		c.logger.WithError(err).Errorf("loadEndpoints(): error decoding")
		return eps
	}

	// decoding from a blank or invalid file
	if eps.Ends == nil || eps.Bans == nil {
		return NewEndpoints()
	}

	eps.Cleanup(c.net.conf.PersistAgeLimit)
	c.logger.Debugf("%d endpoints found", eps.Total())
	return eps
}
