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

var pmLogger = packageLogger.WithField("subpack", "controller")

// controller is responsible for managing Peers and Endpoints
type controller struct {
	net *Network

	peerStatus chan peerStatus
	peerData   chan peerParcel

	stopPeers  chan bool
	stopData   chan bool
	stopOnline chan bool

	peers      *PeerStore
	endpoints  *util.Endpoints
	dialer     *util.Dialer
	listener   *util.LimitedListener
	special    map[string]bool
	specialMtx sync.RWMutex

	lastPeerDial    time.Time
	lastSeedRefresh time.Time
	lastPersist     time.Time

	logger *log.Entry
}

// newController creates a new peer manager for the given controller
// configuration is shared between the two
func newController(network *Network) *controller {
	c := &controller{}
	c.net = network
	conf := network.conf

	c.logger = pmLogger.WithFields(log.Fields{
		"node":    conf.NodeName,
		"port":    conf.ListenPort,
		"network": conf.Network})
	c.logger.WithField("peermanager_init", c).Debugf("Initializing Peer Manager")

	c.dialer = util.NewDialer(conf.BindIP, conf.RedialInterval, conf.RedialReset, conf.DialTimeout, conf.RedialAttempts)
	c.lastPersist = time.Now()

	c.peerStatus = make(chan peerStatus, 10) // TODO reconsider this value
	c.peerData = make(chan peerParcel, c.net.conf.ChannelCapacity)

	c.stopPeers = make(chan bool, 1)
	c.stopData = make(chan bool, 1)
	c.stopOnline = make(chan bool, 1)

	c.bootStrapPeers()
	c.special = make(map[string]bool)
	c.parseSpecial(c.net.conf.Special)

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
		c.endpoints.Ban(peer.IP.Address, time.Now().Add(duration))
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

func (c *controller) parseSpecial(raw string) {
	c.specialMtx.Lock()
	defer c.specialMtx.Unlock()

	if len(raw) == 0 {
		return
	}
	split := strings.Split(raw, ",")
	for _, e := range split {
		ip := net.ParseIP(e)
		if ip == nil {
			c.logger.Warnf("Unable to parse IP in special configuration: %s", e)
		} else {
			c.special[e] = true
		}
	}
}

// Start starts the peer manager
// reads from the seed and connect to peers
func (c *controller) Start() {
	c.logger.Info("Starting the Peer Manager")

	go c.managePeers()
	go c.manageData()
	go c.manageOnline()
	go c.listen()
}

// Stop shuts down the peer manager and all active connections
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
				}
				err := c.peers.Add(pc.peer)
				if err != nil {
					c.logger.Errorf("Unable to add peer %s to peer store because an old peer still exists", pc.peer)
				}
			} else {

				c.peers.Remove(pc.peer)
				if pc.peer.IsIncoming {
					// lock this connection temporarily so we don't try to connect to it
					// before they can reconnect
					c.endpoints.Lock(pc.peer.IP, c.net.conf.DisconnectLock)
				}
			}
			if c.net.prom != nil {
				c.net.prom.Connections.Set(float64(c.peers.Total()))
				c.net.prom.Unique.Set(float64(c.peers.Unique()))
				c.net.prom.Incoming.Set(float64(c.peers.Incoming))
				c.net.prom.Outgoing.Set(float64(c.peers.Outgoing))
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
	fmt.Println("MOO")
	list, err := peer.prot.ParsePeerShare(parcel.Payload)
	fmt.Println(string(parcel.Payload))

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
			c.persist()
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

	count := uint(c.peers.Total())
	if want := int(c.net.conf.Outgoing - count); want > 0 {
		var filtered []util.IP
		var special []util.IP
		c.specialMtx.RLock()
		for _, ip := range c.endpoints.IPs() {
			if c.special[ip.Address] {
				if !c.peers.IsConnected(ip.Address) {
					special = append(special, ip)
				}
			} else if c.dialer.CanDial(ip) {
				filtered = append(filtered, ip)
			}
		}
		c.specialMtx.RUnlock()

		c.logger.Debugf("special: %d, filtered: %d", len(special), len(filtered))
		//fmt.Println(filtered)

		if len(special) > 0 {
			for _, ip := range special {
				go c.Dial(ip)
			}
		}
		if len(filtered) > 0 {
			ips := c.getOutgoingSelection(filtered, want)
			limit := c.net.conf.PeerIPLimitOutgoing
			for _, ip := range ips {

				if c.endpoints.Banned(ip.Address) {
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
	c.specialMtx.RLock()
	defer c.specialMtx.RUnlock()
	if c.special[addr] { // always allow special
		return nil
	}

	if c.endpoints.Banned(addr) {
		return fmt.Errorf("Address %s is banned", addr)
	}

	if uint(c.peers.Total()) >= c.net.conf.Incoming {
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

	ip, err := util.NewIP(addr, "")
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

	peer := NewPeer(c.net, c.peerStatus, c.peerData)
	if ok, err := peer.StartWithHandshake(ip, con, true); ok {
		c.logger.Debugf("Incoming handshake success for peer %s", peer.Hash)
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

	peer := NewPeer(c.net, c.peerStatus, c.peerData)
	if !c.endpoints.IsLocked(ip) {
		c.endpoints.Lock(ip, c.net.conf.HandshakeTimeout)
	}
	if ok, err := peer.StartWithHandshake(ip, con, false); ok {
		c.logger.Debugf("Handshake success for peer %s", peer.Hash)
		c.endpoints.Refresh(peer.IP)
		c.dialer.Reset(peer.IP)
	} else if err.Error() == "connected to ourselves" {
		c.logger.Debugf("Banning ourselves for 1 year")
		c.endpoints.Ban(ip.Address, time.Now().AddDate(50, 0, 0)) // ban for 50 years
		peer.Stop(false)
	} else {
		c.logger.WithError(err).Debugf("Handshake fail with %s", ip)
		peer.Stop(false)
	}
}

func (c *controller) Broadcast(parcel *Parcel, full bool) {
	if full {
		for _, p := range c.peers.Slice() {
			p.Send(parcel)
		}
		return
	}
	// fanout
	selection := c.selectRandomPeers(c.net.conf.Fanout)
	//fmt.Println("selected", len(selection), "random peers")
	for _, p := range selection {
		//fmt.Println("sending to random peer", p.String())
		p.Send(parcel)
	}
	// TODO always send to special
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

	util.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	// TODO add special?
	return peers[:count]
}

// ToPeer sends a parcel to a single peer, specified by their peer hash
//
// If the hash is empty, a random connected peer will be chosen
func (c *controller) ToPeer(hash string, parcel *Parcel) {
	if hash == "" {
		if random := c.selectRandomPeers(1); len(random) > 0 {
			random[0].Send(parcel)
		}
	} else {
		p := c.peers.Get(hash)
		if p != nil {
			p.Send(parcel)
		}
	}
}

func (c *controller) persist() {
	path := c.net.conf.PeerFile
	if path == "" {
		return
	}

	file, err := os.Create(path)
	if err != nil {
		c.logger.WithError(err).Errorf("persist(): File create error for %s", path)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	persist, err := c.endpoints.Persist(c.net.conf.PeerAgeLimit)
	if err != nil {
		c.logger.WithError(err).Error("persist(): Unable to encode endpoints")
		return
	}

	n, err := writer.Write(persist)
	if err != nil {
		c.logger.WithError(err).Errorf("persist(): Unable to write to file, wrote %d bytes", n)
		return
	}

	err = writer.Flush()
	if err != nil {
		c.logger.WithError(err).Error("persist(): Unable to flush")
		return
	}

	c.logger.Debugf("Persisted peers json file")
}

func (c *controller) loadEndpoints() *util.Endpoints {
	path := c.net.conf.PeerFile
	if path == "" {
		return util.NewEndpoints()
	}
	file, err := os.Open(path)
	if err != nil {
		c.logger.WithError(err).Errorf("loadEndpoints(): file open error for %s", path)
		return util.NewEndpoints()
	}

	var eps util.Endpoints
	dec := json.NewDecoder(bufio.NewReader(file))
	err = dec.Decode(&eps)

	if err != nil {
		c.logger.WithError(err).Errorf("loadEndpoints(): error decoding")
		return util.NewEndpoints()
	}

	// decoding from a blank or invalid file
	if eps.Ends == nil || eps.Bans == nil {
		return util.NewEndpoints()
	}

	eps.Cleanup(c.net.conf.PeerAgeLimit)

	return &eps
}

// listen listens for incoming TCP connections and passes them off to peer manager
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
