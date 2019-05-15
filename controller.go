package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/whosoup/factom-p2p/util"
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

	peers     *PeerStore
	endpoints *util.Endpoints
	dialer    *util.Dialer
	listener  *util.LimitedListener

	specialMtx   sync.RWMutex
	specialCount int

	lastPeerDial    time.Time
	lastSeedRefresh time.Time
	lastPersist     time.Time

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

	c.dialer = util.NewDialer(conf.BindIP, conf.RedialInterval, conf.RedialReset, conf.DialTimeout, conf.RedialAttempts)
	c.lastPersist = time.Now()

	c.peerStatus = make(chan peerStatus, 10) // TODO reconsider this value
	c.peerData = make(chan peerParcel, c.net.conf.ChannelCapacity)

	c.stopPeers = make(chan bool, 1)
	c.stopData = make(chan bool, 1)
	c.stopOnline = make(chan bool, 1)

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

func (c *controller) parseSpecial(raw string) []util.IP {
	var ips []util.IP
	split := strings.Split(raw, ",")
	for _, item := range split {
		ip, err := util.ParseAddress(item)
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
}

// Stop shuts down the controller and all active connections
func (c *controller) Stop() {
	c.stopData <- true
	c.stopPeers <- true
	c.stopOnline <- true

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
	c.discoverSeeds()
}

func (c *controller) manageOnline() {
	c.logger.Debug("Start manageOnline()")
	defer c.logger.Debug("Stop manageOnline()")
	for {
		select {
		case <-c.stopOnline:
			return
		case pc := <-c.peerStatus:
			if pc.online {
				old := c.peers.Get(pc.peer.Hash)
				if old != nil {
					old.Stop(true)
					c.peers.Remove(old)
					c.endpoints.RemoveConnection(old.IP)
				}
				err := c.peers.Add(pc.peer)
				c.endpoints.AddConnection(pc.peer.IP)
				if err != nil {
					c.logger.Errorf("Unable to add peer %s to peer store because an old peer still exists", pc.peer)
				}
			} else {
				c.peers.Remove(pc.peer)
				c.endpoints.RemoveConnection(pc.peer.IP)
				if pc.peer.IsIncoming {
					// lock this connection temporarily so we don't try to connect to it
					// before they can reconnect
					c.endpoints.Lock(pc.peer.IP, c.net.conf.DisconnectLock)
				}
			}
			if c.net.prom != nil {
				c.net.prom.Connections.Set(float64(c.peers.Total()))
				c.net.prom.Unique.Set(float64(c.peers.Unique()))
				c.net.prom.Incoming.Set(float64(c.peers.Incoming()))
				c.net.prom.Outgoing.Set(float64(c.peers.Outgoing()))
			}
		}
	}
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
					peer.lastPeerSend = time.Now().Add(-time.Second * 5) // leeway
					go c.sharePeers(peer)
				} else {
					c.logger.Warnf("Peer %s requested peer share sooner than expected", peer)
				}
			case TypePeerResponse:
				if peer.peerShareAsk {
					peer.peerShareAsk = false
					go c.processPeers(peer, parcel)
				} else {
					c.logger.Warnf("Peer %s sent us an umprompted peer share", peer)
				}
			default:
				//not handled
			}
		}
	}

}

func (c *controller) discoverSeeds() {
	c.logger.Info("Contacting seed URL to get peers")
	resp, err := http.Get(c.net.conf.SeedURL)
	if nil != err {
		c.logger.Errorf("discoverSeeds from %s produced error %+v", c.net.conf.SeedURL, err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	report := ""
	for scanner.Scan() {
		line := scanner.Text()
		report = report + "," + line

		address, port, err := net.SplitHostPort(line)
		if err == nil {
			if address == c.net.conf.BindIP && port == c.net.conf.ListenPort {
				c.logger.Debugf("Discovered ourself in seed list")
				continue
			}

			if ip, err := util.NewIP(address, port); err != nil {
				c.logger.WithError(err).Debugf("Invalid endpoint in seed list: %s", line)
			} else {
				c.endpoints.Register(ip, "Seed")
			}

		} else {
			c.logger.Errorf("Bad peer in " + c.net.conf.SeedURL + " [" + line + "]")
		}
	}

	c.logger.Debugf("discoverSeed got peers: %s", report)
}

// processPeers processes a peer share response
func (c *controller) processPeers(peer *Peer, parcel *Parcel) {
	list, err := peer.prot.ParsePeerShare(parcel.Payload)

	if err != nil {
		c.logger.WithError(err).Warnf("Failed to unmarshal peer share from peer %s", peer)
	}

	c.logger.Debugf("Received peer share from %s: %v", peer, list)

	// cycles through list twice but we don't want to add any if one of them is bad
	for _, p := range list {
		if !p.Verify() {
			c.logger.Infof("Peer %s tried to send us peer share with bad data: %s", peer, p)
			return
		}
	}

	for _, p := range list {
		ip, err := util.NewIP(p.Address, p.Port)
		if err != nil {
			c.logger.WithError(err).Infof("Unable to register endpoint %s:%s from peer %s", p.Address, p.Port, peer)
		} else {
			c.endpoints.Register(ip, peer.IP.Address)
		}
	}

	if c.net.prom != nil {
		c.net.prom.KnownPeers.Set(float64(c.endpoints.Total()))
	}
}

// sharePeers creates a list of peers to share and sends it to peer
func (c *controller) sharePeers(peer *Peer) {
	var list []util.IP
	for _, ip := range c.endpoints.IPs() {
		if ip != peer.IP && !c.dialer.Failed(ip) {
			list = append(list, ip)
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

// managePeers is responsible for everything that involves proactive management
// not based on reactions. runs once a second
func (c *controller) managePeers() {
	c.logger.Debug("Start managePeers()")
	defer c.logger.Debug("Stop managePeers()")

	for {
		if time.Since(c.lastPersist) > c.net.conf.PersistInterval {
			c.lastPersist = time.Now()
			err := c.endpoints.Persist(c.net.conf.PersistFile, c.net.conf.PersistLevel, c.net.conf.PersistMinimum, c.net.conf.PersistAgeLimit)
			if err != nil {
				c.logger.WithError(err).Errorf("unable to persist peers")
			}
		}

		if time.Since(c.lastSeedRefresh) > c.net.conf.PeerReseedInterval {
			c.lastSeedRefresh = time.Now()
			c.discoverSeeds()
		}

		// manage online connectivity
		if time.Since(c.lastPeerDial) > c.net.conf.RedialInterval {
			c.lastPeerDial = time.Now()
			c.managePeersDialOutgoing()
		}

		metrics := make(map[string]PeerMetrics)
		for _, p := range c.peers.Slice() {
			metrics[p.Hash] = p.GetMetrics()

			if time.Since(p.lastPeerRequest) > c.net.conf.PeerRequestInterval {
				p.lastPeerRequest = time.Now()
				p.peerShareAsk = true
				c.logger.Debugf("Requesting peers from %s", p.Hash)
				req := newParcel(TypePeerRequest, []byte("Peer Request"))
				p.Send(req)
			}

			if time.Since(p.LastSend) > c.net.conf.PingInterval {
				ping := newParcel(TypePing, []byte("Ping"))
				p.Send(ping)
			}
		}

		if c.net.metricsHook != nil {
			go c.net.metricsHook(metrics)
		}

		// manages peers every second
		select {
		case <-c.stopPeers:
			return
		case <-time.After(time.Second):
		}
	}
}

func (c *controller) managePeersDialOutgoing() {
	c.logger.Debugf("We have %d peers online or connecting", c.peers.Total())

	c.specialMtx.RLock()
	defer c.specialMtx.RUnlock()
	count := c.peers.Total()
	if want := int(c.net.conf.Outgoing - uint(count-c.specialCount)); want > 0 || c.specialCount > 0 {
		var filtered []util.IP
		var special []util.IP
		for _, ip := range c.endpoints.IPs() {
			if c.endpoints.IsSpecial(ip) {
				if !c.peers.IsConnected(ip.Address) {
					special = append(special, ip)
				}
			} else if c.dialer.CanDial(ip) {
				filtered = append(filtered, ip)
			}
		}

		c.logger.Debugf("special: %d, filtered: %d", len(special), len(filtered))
		//fmt.Println(filtered)

		if len(special) > 0 {
			for _, ip := range special {
				if c.endpoints.IsLocked(ip) {
					continue
				}
				go c.Dial(ip)
			}
		}
		if len(filtered) > 0 {
			ips := c.getOutgoingSelection(filtered, want)
			limit := c.net.conf.PeerIPLimitOutgoing
			for _, ip := range ips {

				if c.endpoints.BannedEndpoint(ip) {
					continue
				}
				if limit > 0 && uint(c.peers.Count(ip.Address)) >= limit {
					continue
				}
				if c.endpoints.IsLocked(ip) {
					continue
				}
				go c.Dial(ip)
			}
		}
	}
}

func (c *controller) allowIncoming(addr string) error {
	if c.endpoints.BannedAddress(addr) {
		return fmt.Errorf("Address %s is banned", addr)
	}

	if uint(c.peers.Total()) >= c.net.conf.Incoming && !c.endpoints.IsSpecialAddress(addr) {
		return fmt.Errorf("Refusing incoming connection from %s because we are maxed out (%d of %d)", addr, c.peers.Total(), c.net.conf.Incoming)
	}

	if c.net.conf.PeerIPLimitIncoming > 0 && uint(c.peers.Count(addr)) >= c.net.conf.PeerIPLimitIncoming {
		return fmt.Errorf("Rejecting %s due to per ip limit of %d", addr, c.net.conf.PeerIPLimitIncoming)
	}

	return nil
}

func (c *controller) handleIncoming(con net.Conn) {
	if c.net.prom != nil {
		c.net.prom.Connecting.Inc()
		defer c.net.prom.Connecting.Dec()
	}

	addr, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		c.logger.WithError(err).Debugf("Unable to parse address %s", con.RemoteAddr().String())
		con.Close()
		return
	}

	// port is overriden during handshake, use default port as temp port
	ip, err := util.NewIP(addr, c.net.conf.ListenPort)
	if err != nil { // should never happen for incoming
		c.logger.WithError(err).Debugf("Unable to decode address %s", addr)
		con.Close()
		return
	}

	if err = c.allowIncoming(addr); err != nil {
		c.logger.WithError(err).Infof("Rejecting connection")
		con.Close()
		return
	}

	peer := newPeer(c.net, c.peerStatus, c.peerData)
	if ok, err := peer.StartWithHandshake(ip, con, true); ok {
		c.logger.Debugf("Incoming handshake success for peer %s", peer.Hash)

		if c.endpoints.BannedEndpoint(peer.IP) {
			c.logger.Debugf("Peer %s is banned, disconnecting", peer.Hash)
			return
		}

		c.endpoints.Register(peer.IP, "Incoming")
		c.endpoints.Lock(peer.IP, time.Hour*8760*50) // 50 years
		c.dialer.Reset(peer.IP)
	} else {
		c.logger.WithError(err).Debugf("Handshake failed for address %s, stopping", ip)
		peer.Stop(false)
	}
}

func (c *controller) Dial(ip util.IP) {
	if c.net.prom != nil {
		c.net.prom.Connecting.Inc()
		defer c.net.prom.Connecting.Dec()
	}

	if ip.Port == "" {
		ip.Port = c.net.conf.ListenPort // TODO add a "default port"?
		c.logger.Debugf("Dialing to %s (with no previously known port)", ip)
	} else {
		c.logger.Debugf("Dialing to %s", ip)
	}

	con, err := c.dialer.Dial(ip)
	if err != nil {
		c.logger.WithError(err).Infof("Failed to dial to %s", ip)
		return
	}

	peer := newPeer(c.net, c.peerStatus, c.peerData)
	if !c.endpoints.IsLocked(ip) {
		c.endpoints.Lock(ip, c.net.conf.HandshakeTimeout)
	}
	if ok, err := peer.StartWithHandshake(ip, con, false); ok {
		c.logger.Debugf("Handshake success for peer %s", peer.Hash)
		c.endpoints.Register(peer.IP, "Dial")
		c.dialer.Reset(peer.IP)
	} else if err.Error() == "loopback" {
		c.logger.Debugf("Banning ourselves for 50 years")
		c.endpoints.BanEndpoint(ip, time.Now().AddDate(50, 0, 0)) // ban for 50 years
		peer.Stop(false)
	} else {
		c.logger.WithError(err).Debugf("Handshake fail with %s", ip)
		peer.Stop(false)
	}
}

// getOutgoingSelection creates a subset of total connectable peers by getting
// as much prefix variation as possible
//
// Takes the input and spreads peers out over n equally sized buckets based on their
// ipv4 prefix, then iterates over those buckets and removes a random peer from each
// one until it has enough
func (c *controller) getOutgoingSelection(filtered []util.IP, wanted int) []util.IP {
	if wanted < 1 {
		return nil
	}
	// we have just enough
	if len(filtered) <= wanted {
		c.logger.Debugf("getOutgoingSelection returning %d peers", len(filtered))
		return filtered
	}

	if wanted == 1 { // edge case
		rand := c.net.rng.Intn(len(filtered))
		return []util.IP{filtered[rand]}
	}

	// generate a list of peers distant to each other
	buckets := make([][]util.IP, wanted)
	bucketSize := uint32(4294967295/uint32(wanted)) + 1 // 33554432 for wanted=128

	// distribute peers over n buckets
	for _, peer := range filtered {
		bucketIndex := int(peer.Location / bucketSize)
		buckets[bucketIndex] = append(buckets[bucketIndex], peer)
	}

	// pick random peers from each bucket
	var picked []util.IP
	for len(picked) < wanted {
		offset := c.net.rng.Intn(len(buckets)) // start at a random point in the bucket array
		for i := 0; i < len(buckets); i++ {
			bi := (i + offset) % len(buckets)
			bucket := buckets[bi]
			if len(bucket) > 0 {
				pi := c.net.rng.Intn(len(bucket)) // random member in bucket
				picked = append(picked, bucket[pi])
				bucket[pi] = bucket[len(bucket)-1] // fast remove
				buckets[bi] = bucket[:len(bucket)-1]
			}
		}
	}

	c.logger.Debugf("getOutgoingSelection returning %d peers: %+v", len(picked), picked)
	return picked
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

	util.Shuffle(len(regular), func(i, j int) {
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

func (c *controller) loadEndpoints() *util.Endpoints {
	eps := util.NewEndpoints()

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
		return util.NewEndpoints()
	}

	eps.Cleanup(c.net.conf.PersistAgeLimit)
	c.logger.Debugf("%d endpoints found", eps.Total())
	return eps
}

// listen listens for incoming TCP connections and passes them off to handshake maneuver
func (c *controller) listen() {
	tmpLogger := c.logger.WithFields(log.Fields{"address": c.net.conf.BindIP, "port": c.net.conf.ListenPort})
	tmpLogger.Debug("controller.listen() starting up")

	addr := fmt.Sprintf("%s:%s", c.net.conf.BindIP, c.net.conf.ListenPort)

	l, err := util.NewLimitedListener(addr, c.net.conf.ListenLimit)
	if err != nil {
		tmpLogger.WithError(err).Error("controller.Start() unable to start limited listener")
		return
	}

	c.listener = l

	// start permanent loop
	// terminates on program exit or when listener is closed
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && !ne.Timeout() {
				if !ne.Temporary() {
					tmpLogger.WithError(err).Warn("controller.acceptLoop() error accepting")
					return
				}
			}
			continue
		}

		go c.handleIncoming(conn)
	}
}
