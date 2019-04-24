package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var pmLogger = packageLogger.WithField("subpack", "peerManager")

// peerManager is responsible for managing all the Peers, both online and offline
type peerManager struct {
	net *Network

	peerDisconnect chan *Peer

	stopPeers  chan bool
	stopData   chan bool
	stopOnline chan bool

	peers     *PeerStore
	endpoints *Endpoints
	dialer    *Dialer
	special   map[string]bool
	//bans      map[string]time.Time

	lastPeerDial    time.Time
	lastSeedRefresh time.Time
	lastPersist     time.Time

	logger *log.Entry
}

// newPeerManager creates a new peer manager for the given controller
// configuration is shared between the two
func newPeerManager(network *Network) *peerManager {
	pm := &peerManager{}
	pm.net = network
	c := network.conf

	pm.logger = pmLogger.WithFields(log.Fields{
		"node":    c.NodeName,
		"port":    c.ListenPort,
		"network": c.Network})
	pm.logger.WithField("peermanager_init", c).Debugf("Initializing Peer Manager")

	pm.dialer = NewDialer(c.BindIP, c.RedialInterval, c.DialTimeout, c.RedialAttempts)
	pm.lastPersist = time.Now()

	pm.peerDisconnect = make(chan *Peer, 10) // TODO reconsider this value

	pm.stopPeers = make(chan bool, 1)
	pm.stopData = make(chan bool, 1)
	pm.stopOnline = make(chan bool, 1)

	pm.bootStrapPeers()
	pm.special = make(map[string]bool)
	pm.parseSpecial(pm.net.conf.Special)

	return pm
}

// ban bans the peer indicated by the hash as well as any other peer from that ip
// address
func (pm *peerManager) ban(hash string, duration time.Duration) {
	peer := pm.peers.Get(hash)
	if peer != nil {
		pm.endpoints.Ban(peer.IP.Address, time.Now().Add(duration))
		for _, p := range pm.peers.Slice() {
			if p.IP.Address == peer.IP.Address {
				peer.Stop(true)
			}
		}
	}
}

func (pm *peerManager) merit(hash string) {
	peer := pm.peers.Get(hash)
	if peer != nil {
		peer.quality(1)
	}
}

func (pm *peerManager) demerit(hash string) {
	peer := pm.peers.Get(hash)
	if peer != nil {
		if peer.quality(-1) < pm.net.conf.MinimumQualityScore {
			pm.ban(hash, pm.net.conf.AutoBan)
		}
	}
}

func (pm *peerManager) disconnect(hash string) {
	peer := pm.peers.Get(hash)
	if peer != nil {
		peer.Stop(true)
	}
}

func (pm *peerManager) parseSpecial(raw string) {
	if len(raw) == 0 {
		return
	}
	split := strings.Split(raw, ",")
	for _, e := range split {
		ip := net.ParseIP(e)
		if ip == nil {
			pm.logger.Warnf("Unable to parse IP in special configuration: %s", e)
		} else {
			pm.special[e] = true
		}
	}
}

// Start starts the peer manager
// reads from the seed and connect to peers
func (pm *peerManager) Start() {
	pm.logger.Info("Starting the Peer Manager")

	// TODO discover from seed
	// 		parse and dial special peers
	//go pm.receiveData()
	go pm.managePeers()
	go pm.manageData()
	go pm.manageOnline()
}

// Stop shuts down the peer manager and all active connections
func (pm *peerManager) Stop() {
	pm.stopData <- true
	pm.stopPeers <- true
	pm.stopOnline <- true

	for _, p := range pm.peers.Slice() {
		p.Stop(false)
		pm.peers.Remove(p)
	}
}

func (pm *peerManager) bootStrapPeers() {
	pm.peers = NewPeerStore()
	pm.endpoints = pm.loadEndpoints()
	if pm.endpoints == nil {
		pm.endpoints = NewEndpoints()
	}
	pm.lastSeedRefresh = time.Now()
	pm.discoverSeeds()
}

func (pm *peerManager) manageOnline() {
	pm.logger.Debug("Start manageOnline()")
	defer pm.logger.Debug("Stop manageOnline()")
	for {
		select {
		case <-pm.stopOnline:
			return
		case p := <-pm.peerDisconnect:
			if p.IsIncoming {
				pm.endpoints.SetConnectionLock(p.IP)
			}
			pm.peers.Remove(p)
		}
	}
}

func (pm *peerManager) manageData() {
	pm.logger.Debug("Start manageData()")
	defer pm.logger.Debug("Stop manageData()")
	for {
		select {
		case <-pm.stopData:
			return
		case data := <-pm.net.peerParcel:
			parcel := data.Parcel
			peer := data.Peer

			pm.logger.Debugf("Received parcel type %s from %s", parcel.MessageType(), peer.Hash)

			switch parcel.Header.Type {
			case TypeMessagePart: // deprecated
			case TypeHeartbeat: // deprecated
			case TypePing:
			case TypePong:
			case TypeAlert:

			case TypeMessage: // Application message, send it on.
				//				ApplicationMessagesReceived++
				pm.net.FromNetwork.Send(parcel)
			case TypePeerRequest:
				if time.Since(peer.lastPeerSend) >= pm.net.conf.PeerRequestInterval {
					peer.lastPeerSend = time.Now()
					go pm.sharePeers(peer)
				} else {
					pm.logger.Warnf("Peer %s requested peer share sooner than expected", peer)
				}
			case TypePeerResponse:
				if peer.peerShareAsk {
					peer.peerShareAsk = false
					go pm.processPeers(peer, parcel)
				} else {
					pm.logger.Warnf("Peer %s sent us an umprompted peer share", peer)
				}
			default:
				pm.logger.Warnf("Peer %s sent unknown parcel.Header.Type?: %+v ", peer, parcel)
			}
		}
	}

}

func (pm *peerManager) discoverSeeds() {
	pm.logger.Info("Contacting seed URL to get peers")
	resp, err := http.Get(pm.net.conf.SeedURL)
	if nil != err {
		pm.logger.Errorf("discoverSeeds from %s produced error %+v", pm.net.conf.SeedURL, err)
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
			if address == pm.net.conf.BindIP && port == pm.net.conf.ListenPort {
				pm.logger.Debugf("Discovered ourself in seed list")
				continue
			}

			if ip, err := NewIP(address, port); err != nil {
				pm.logger.WithError(err).Debugf("Invalid endpoint in seed list: %s", line)
			} else {
				pm.endpoints.Register(ip, false, "Seed")
			}

		} else {
			pm.logger.Errorf("Bad peer in " + pm.net.conf.SeedURL + " [" + line + "]")
		}
	}

	pm.logger.Debugf("discoverSeed got peers: %s", report)
}

// processPeers processes a peer share response
func (pm *peerManager) processPeers(peer *Peer, parcel *Parcel) {
	list := make([]PeerShare, 0)
	err := json.Unmarshal(parcel.Payload, &list)
	if err != nil {
		pm.logger.WithError(err).Warnf("Failed to unmarshal peer share from peer %s", peer)
	}

	// cycles through list twice but we don't want to add any if one of them is bad
	for _, p := range list {
		if !p.Verify() {
			pm.logger.Infof("Peer %s tried to send us peer share with bad data: %s", peer, p)
			return
		}
	}

	for _, p := range list {
		ip, err := NewIP(p.Address, p.Port)
		if err != nil {
			pm.logger.WithError(err).Infof("Unable to register endpoint %s:%s from peer %s", p.Address, p.Port, peer)
		} else {
			pm.endpoints.Register(ip, false, peer.IP.Address)
		}
	}
}

// sharePeers creates a list of peers to share and sends it to peer
func (pm *peerManager) sharePeers(peer *Peer) {
	list := pm.filteredSharing(peer)
	pm.logger.Debugf("Sharing peers with %s: %v", peer, list)
	json, ok := json.Marshal(list)
	if ok != nil {
		pm.logger.WithError(ok).Error("Failed to marshal peer list to json")
		return
	}
	parcel := NewParcel(TypePeerResponse, json)
	peer.Send(parcel)
}

func (pm *peerManager) filteredSharing(peer *Peer) []PeerShare {
	var filtered []PeerShare
	for _, ip := range pm.endpoints.IPs() {
		if ip != peer.IP {
			filtered = append(filtered, PeerShare{
				Address:      ip.Address,
				Port:         ip.Port,
				QualityScore: 1,
			})
		}
	}
	return filtered
}

// managePeers is responsible for everything that involves proactive management
// not based on reactions. runs once a second
func (pm *peerManager) managePeers() {
	pm.logger.Debug("Start managePeers()")
	defer pm.logger.Debug("Stop managePeers()")

	for {
		if time.Since(pm.lastPersist) > pm.net.conf.PersistInterval {
			pm.lastPersist = time.Now()
			pm.persist()
		}

		if time.Since(pm.lastSeedRefresh) > pm.net.conf.PeerReseedInterval {
			pm.lastSeedRefresh = time.Now()
			pm.discoverSeeds()
		}

		// manage online connectivity
		if time.Since(pm.lastPeerDial) > pm.net.conf.RedialInterval {
			pm.lastPeerDial = time.Now()
			pm.managePeersDialOutgoing()
		}

		metrics := make(map[string]PeerMetrics)
		for _, p := range pm.peers.Slice() {
			metrics[p.Hash] = p.GetMetrics()

			if time.Since(p.lastPeerRequest) > pm.net.conf.PeerRequestInterval {
				p.lastPeerRequest = time.Now()
				p.peerShareAsk = true
				pm.logger.Debugf("Requesting peers from %s", p.Hash)
				req := NewParcel(TypePeerRequest, []byte("Peer Request"))
				p.Send(req)
			}

			if time.Since(p.LastSend) > pm.net.conf.PingInterval {
				ping := NewParcel(TypePing, []byte("Ping"))
				p.Send(ping)
			}
		}

		if pm.net.metricsHook != nil {
			go pm.net.metricsHook(metrics)
		}

		// manages peers every second

		select {
		case <-pm.stopPeers:
			return
		case <-time.After(time.Second):
		}

	}
}

func (pm *peerManager) managePeersDialOutgoing() {
	pm.logger.Debugf("We have %d peers online or connecting", pm.peers.Total())

	count := uint(pm.peers.Total())
	if want := int(pm.net.conf.Outgoing - count); want > 0 {
		var filtered []IP
		var special []IP
		for _, ip := range pm.endpoints.IPs() {
			if pm.special[ip.Address] {
				if !pm.peers.IsConnected(ip.Address) {
					special = append(special, ip)
				}
			} else if pm.dialer.CanDial(ip) && pm.endpoints.ConnectionLock(ip) > pm.net.conf.DisconnectLock {
				filtered = append(filtered, ip)
			}
		}

		pm.logger.Debugf("special: %d, filtered: %d", len(special), len(filtered))

		if len(special) > 0 {
			for _, ip := range special {
				go pm.Dial(ip)
			}
		}
		if len(filtered) > 0 {
			ips := pm.getOutgoingSelection(filtered, want)
			limit := pm.net.conf.PeerIPLimitOutgoing
			for _, ip := range ips {
				if limit > 0 && uint(pm.peers.Count(ip.Address)) >= limit {
					continue
				}
				go pm.Dial(ip)
			}
		}
	}
}

func (pm *peerManager) allowIncoming(addr string) error {
	if pm.special[addr] { // always allow special
		return nil
	}

	if pm.endpoints.Banned(addr) {
		return fmt.Errorf("Address %s is banned", addr)
	}

	if pm.net.conf.RefuseIncoming {
		return fmt.Errorf("Refusing all incoming connections")
	}

	if uint(pm.peers.Total()) >= pm.net.conf.Incoming {
		return fmt.Errorf("Refusing incoming connection from %s because we are maxed out", addr)
	}

	if pm.net.conf.PeerIPLimitIncoming > 0 && uint(pm.peers.Count(addr)) >= pm.net.conf.PeerIPLimitIncoming {
		return fmt.Errorf("Rejecting %s due to per ip limit of %d", addr, pm.net.conf.PeerIPLimitIncoming)
	}

	return nil
}

func (pm *peerManager) HandleIncoming(con net.Conn) {
	addr, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		pm.logger.WithError(err).Debugf("Unable to parse address %s", con.RemoteAddr().String())
		con.Close()
		return
	}

	ip, err := NewIP(addr, "")
	if err != nil { // should never happen for incoming
		pm.logger.WithError(err).Debugf("Unable to decode address %s", addr)
		con.Close()
		return
	}

	if err = pm.allowIncoming(addr); err != nil {
		pm.logger.WithError(err).Infof("Rejecting connection")
		con.Close()
		return
	}

	// TODO limit by endpoint
	peer := NewPeer(pm.net, pm.peerDisconnect)
	if peer.StartWithHandshake(ip, con, true) {
		old := pm.peers.Replace(peer)

		pm.endpoints.Register(peer.IP, false, "Incoming")
		if old != nil {
			old.Stop(false)
		}
		pm.dialer.Reset(peer.IP)
	}
}

func (pm *peerManager) Dial(ip IP) {
	if ip.Port == "" {
		ip.Port = pm.net.conf.ListenPort // TODO add a "default port"?
		pm.logger.Debugf("Dialing to %s (with no previously known port)", ip)
	} else {
		pm.logger.Debugf("Dialing to %s", ip)
	}

	con, err := pm.dialer.Dial(ip)
	if err != nil {
		pm.logger.WithError(err).Infof("Failed to dial to %s", ip)
		return
	}

	peer := NewPeer(pm.net, pm.peerDisconnect)
	if peer.StartWithHandshake(ip, con, false) {
		if old := pm.peers.Replace(peer); old != nil {
			old.Stop(false)
		}

		pm.endpoints.Refresh(peer.IP)
		pm.dialer.Reset(peer.IP)
	}
}

func (pm *peerManager) Broadcast(parcel *Parcel, full bool) {
	if full {
		for _, p := range pm.peers.Slice() {
			p.Send(parcel)
		}
		return
	}
	// fanout
	selection := pm.selectRandomPeers(pm.net.conf.Fanout)
	fmt.Println("selected", len(selection))
	for _, p := range selection {
		fmt.Println("sending to random peer", p.String())
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
func (pm *peerManager) getOutgoingSelection(filtered []IP, wanted int) []IP {
	if wanted < 1 {
		return nil
	}
	// we have just enough
	if len(filtered) <= wanted {
		pm.logger.Debugf("getOutgoingSelection returning %d peers", len(filtered))
		return filtered
	}

	if wanted == 1 { // edge case
		rand := pm.net.rng.Intn(len(filtered))
		return []IP{filtered[rand]}
	}

	// generate a list of peers distant to each other
	buckets := make([][]IP, wanted)
	bucketSize := uint32(4294967295/uint32(wanted)) + 1 // 33554432 for wanted=128

	// distribute peers over n buckets
	for _, peer := range filtered {
		bucketIndex := int(peer.Location / bucketSize)
		buckets[bucketIndex] = append(buckets[bucketIndex], peer)
	}

	// pick random peers from each bucket
	var picked []IP
	for len(picked) < wanted {
		offset := pm.net.rng.Intn(len(buckets)) // start at a random point in the bucket array
		for i := 0; i < len(buckets); i++ {
			bi := (i + offset) % len(buckets)
			bucket := buckets[bi]
			if len(bucket) > 0 {
				pi := pm.net.rng.Intn(len(bucket)) // random member in bucket
				picked = append(picked, bucket[pi])
				bucket[pi] = bucket[len(bucket)-1] // fast remove
				buckets[bi] = bucket[:len(bucket)-1]
			}
		}
	}

	pm.logger.Debugf("getOutgoingSelection returning %d peers: %+v", len(picked), picked)
	return picked
}

func (pm *peerManager) selectRandomPeers(count uint) []*Peer {
	peers := pm.peers.Slice()

	// not enough to randomize
	if uint(len(peers)) <= count {
		return peers
	}

	shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	// TODO add special?
	return peers[:count]
}

// ToPeer sends a parcel to a single peer, specified by their peer hash
//
// If the hash is empty, a random connected peer will be chosen
func (pm *peerManager) ToPeer(hash string, parcel *Parcel) {
	if hash == "" {
		if random := pm.selectRandomPeers(1); len(random) > 0 {
			random[0].Send(parcel)
		}
	} else {
		p := pm.peers.Get(hash)
		if p != nil {
			p.Send(parcel)
		}
	}
}

func (pm *peerManager) persist() {
	path := pm.net.conf.PeerFile
	if path == "" {
		return
	}

	file, err := os.Create(path)
	if err != nil {
		pm.logger.WithError(err).Errorf("persist(): File create error for %s", path)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	persist, err := pm.endpoints.Persist()
	if err != nil {
		pm.logger.WithError(err).Error("persist(): Unable to encode endpoints")
		return
	}

	n, err := writer.Write(persist)
	if err != nil {
		pm.logger.WithError(err).Error("persist(): Unable to write to file, wrote %d bytes", n)
		return
	}

	err = writer.Flush()
	if err != nil {
		pm.logger.WithError(err).Error("persist(): Unable to flush")
		return
	}

	pm.logger.Debugf("Persisted peers json file")
}

func (pm *peerManager) loadEndpoints() *Endpoints {
	path := pm.net.conf.PeerFile
	if path == "" {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		pm.logger.WithError(err).Errorf("loadEndpoints(): file open error for %s", path)
		return nil
	}

	var eps Endpoints
	dec := json.NewDecoder(bufio.NewReader(file))
	err = dec.Decode(&eps)

	if err != nil {
		pm.logger.WithError(err).Errorf("loadEndpoints(): error decoding")
		return nil
	}

	return &eps
}
