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

var pmLogger = packageLogger.WithField("subpack", "peerManager")

// peerManager is responsible for managing all the Peers, both online and offline
type peerManager struct {
	net *Network

	peerStatus chan peerStatus

	stopPeers  chan bool
	stopData   chan bool
	stopOnline chan bool

	peers      *PeerStore
	endpoints  *util.Endpoints
	dialer     *util.Dialer
	special    map[string]bool
	specialMtx sync.RWMutex
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

	pm.dialer = util.NewDialer(c.BindIP, c.RedialInterval, c.DialTimeout, c.RedialAttempts)
	pm.lastPersist = time.Now()

	pm.peerStatus = make(chan peerStatus, 10) // TODO reconsider this value

	pm.stopPeers = make(chan bool, 1)
	pm.stopData = make(chan bool, 1)
	pm.stopOnline = make(chan bool, 1)

	pm.bootStrapPeers()
	pm.special = make(map[string]bool)
	pm.parseSpecial(pm.net.conf.Special)

	pm.net.prom.KnownPeers.Set(float64(pm.endpoints.Total()))

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
	pm.specialMtx.Lock()
	defer pm.specialMtx.Unlock()
	pm.special = make(map[string]bool)

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
	pm.endpoints = pm.loadEndpoints() // creates blank if none exist
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
		case pc := <-pm.peerStatus:
			if pc.online {
				old := pm.peers.Get(pc.peer.Hash)
				if old != nil {
					old.Stop(true)
					pm.peers.Remove(old)
				}
				err := pm.peers.Add(pc.peer)
				if err != nil {
					pm.logger.Errorf("Unable to add peer %s to peer store because an old peer still exists", pc.peer)
				}
			} else {

				pm.peers.Remove(pc.peer)
				if pc.peer.IsIncoming {
					// lock this connection temporarily so we don't try to connect to it
					// before they can reconnect
					pm.endpoints.Lock(pc.peer.IP, pm.net.conf.DisconnectLock)
				}
			}

			pm.net.prom.Connections.Set(float64(pm.peers.Total()))
			pm.net.prom.Unique.Set(float64(pm.peers.Unique()))
			pm.net.prom.Incoming.Set(float64(pm.peers.Incoming))
			pm.net.prom.Outgoing.Set(float64(pm.peers.Outgoing))
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

			if err := parcel.Valid(); err != nil {
				pm.logger.WithError(err).Warnf("received invalid parcel, disconnecting peer %s", peer)
				peer.Stop(true)
				pm.net.prom.Invalid.Inc()
				continue
			}
			pm.logger.Debugf("Received parcel %s from %s", parcel, peer)
			pm.net.prom.ParcelsReceived.Inc()

			switch parcel.Header.Type {
			case TypePing:
				go func() {
					parcel := newParcel(TypePong, []byte("Pong"))
					peer.Send(parcel)
				}()
			case TypeMessage:
				//pm.net.FromNetwork.Send(parcel)
				fallthrough
			case TypeMessagePart:
				pm.net.prom.AppReceived.Inc()
				parcel.Header.Type = TypeMessage
				pm.net.FromNetwork.Send(parcel)
			case TypePeerRequest:
				if time.Since(peer.lastPeerSend) >= pm.net.conf.PeerRequestInterval {
					peer.lastPeerSend = time.Now().Add(-time.Second * 5) // leeway
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
				//not handled
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

			if ip, err := util.NewIP(address, port); err != nil {
				pm.logger.WithError(err).Debugf("Invalid endpoint in seed list: %s", line)
			} else {
				pm.endpoints.Register(ip, "Seed")
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
		ip, err := util.NewIP(p.Address, p.Port)
		if err != nil {
			pm.logger.WithError(err).Infof("Unable to register endpoint %s:%s from peer %s", p.Address, p.Port, peer)
		} else {
			pm.endpoints.Register(ip, peer.IP.Address)
		}
	}

	pm.net.prom.KnownPeers.Set(float64(pm.endpoints.Total()))
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
	parcel := newParcel(TypePeerResponse, json)
	peer.Send(parcel)
}

func (pm *peerManager) filteredSharing(peer *Peer) []PeerShare {
	var filtered []PeerShare
	src := make(map[string]time.Time)
	for _, ip := range pm.endpoints.IPs() {
		if ip != peer.IP {
			filtered = append(filtered, PeerShare{
				Address:      ip.Address,
				Port:         ip.Port,
				QualityScore: peer.QualityScore,
				NodeID:       peer.NodeID,
				Hash:         peer.Hash,
				Location:     peer.IP.Location,
				Network:      pm.net.conf.Network,
				Type:         0,
				Connections:  1,
				LastContact:  peer.LastReceive,
				Source:       src,
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
				req := newParcel(TypePeerRequest, []byte("Peer Request"))
				p.Send(req)
			}

			if time.Since(p.LastSend) > pm.net.conf.PingInterval {
				ping := newParcel(TypePing, []byte("Ping"))
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
		var filtered []util.IP
		var special []util.IP
		pm.specialMtx.RLock()
		for _, ip := range pm.endpoints.IPs() {
			if pm.special[ip.Address] {
				if !pm.peers.IsConnected(ip.Address) {
					special = append(special, ip)
				}
			} else if pm.dialer.CanDial(ip) {
				filtered = append(filtered, ip)
			}
		}
		pm.specialMtx.RUnlock()

		pm.logger.Debugf("special: %d, filtered: %d", len(special), len(filtered))
		//fmt.Println(filtered)

		if len(special) > 0 {
			for _, ip := range special {
				go pm.Dial(ip)
			}
		}
		if len(filtered) > 0 {
			ips := pm.getOutgoingSelection(filtered, want)
			limit := pm.net.conf.PeerIPLimitOutgoing
			for _, ip := range ips {

				if pm.endpoints.Banned(ip.Address) {
					continue
				}
				if limit > 0 && uint(pm.peers.Count(ip.Address)) >= limit {
					continue
				}
				if pm.endpoints.IsLocked(ip) {
					continue
				}
				go pm.Dial(ip)
			}
		}
	}
}

func (pm *peerManager) allowIncoming(addr string) error {
	pm.specialMtx.RLock()
	defer pm.specialMtx.RUnlock()
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
	pm.net.prom.Connecting.Inc()
	defer pm.net.prom.Connecting.Dec()

	addr, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		pm.logger.WithError(err).Debugf("Unable to parse address %s", con.RemoteAddr().String())
		con.Close()
		return
	}

	ip, err := util.NewIP(addr, "")
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

	peer := NewPeer(pm.net, pm.peerStatus)
	if ok, err := peer.StartWithHandshake(ip, con, true); ok {
		pm.logger.Debug("Incoming handshake success for peer %s", peer.Hash)
		pm.endpoints.Register(peer.IP, "Incoming")
		pm.endpoints.Lock(peer.IP, time.Hour*8760*50) // 50 years
		pm.dialer.Reset(peer.IP)
	} else {
		pm.logger.WithError(err).Debugf("Handshake failed for address %s, stopping", ip)
		peer.Stop(false)
	}
}

func (pm *peerManager) Dial(ip util.IP) {
	pm.net.prom.Connecting.Inc()
	defer pm.net.prom.Connecting.Dec()

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

	peer := NewPeer(pm.net, pm.peerStatus)
	if !pm.endpoints.IsLocked(ip) {
		pm.endpoints.Lock(ip, pm.net.conf.HandshakeTimeout)
	}
	if ok, err := peer.StartWithHandshake(ip, con, false); ok {
		pm.logger.Debugf("Handshake success for peer %s", peer.Hash)
		pm.endpoints.Refresh(peer.IP)
		pm.dialer.Reset(peer.IP)
	} else if err.Error() == "connected to ourselves" {
		pm.logger.Debugf("Banning ourselves for 1 year")
		pm.endpoints.Ban(ip.Address, time.Now().AddDate(50, 0, 0)) // ban for 50 years
		peer.Stop(false)
	} else {
		pm.logger.WithError(err).Debugf("Handshake fail with %s", ip)
		peer.Stop(false)
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
func (pm *peerManager) getOutgoingSelection(filtered []util.IP, wanted int) []util.IP {
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

	util.Shuffle(len(peers), func(i, j int) {
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

	persist, err := pm.endpoints.Persist(pm.net.conf.PeerAgeLimit)
	if err != nil {
		pm.logger.WithError(err).Error("persist(): Unable to encode endpoints")
		return
	}

	n, err := writer.Write(persist)
	if err != nil {
		pm.logger.WithError(err).Errorf("persist(): Unable to write to file, wrote %d bytes", n)
		return
	}

	err = writer.Flush()
	if err != nil {
		pm.logger.WithError(err).Error("persist(): Unable to flush")
		return
	}

	pm.logger.Debugf("Persisted peers json file")
}

func (pm *peerManager) loadEndpoints() *util.Endpoints {
	path := pm.net.conf.PeerFile
	if path == "" {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		pm.logger.WithError(err).Errorf("loadEndpoints(): file open error for %s", path)
		return nil
	}

	var eps util.Endpoints
	dec := json.NewDecoder(bufio.NewReader(file))
	err = dec.Decode(&eps)

	if err != nil {
		pm.logger.WithError(err).Errorf("loadEndpoints(): error decoding")
		return nil
	}

	// decoding from a blank or invalid file
	if eps.Ends == nil || eps.Bans == nil {
		return util.NewEndpoints()
	}

	eps.Cleanup(pm.net.conf.PeerAgeLimit)

	return &eps
}
