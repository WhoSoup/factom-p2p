package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var pmLogger = packageLogger.WithField("subpack", "peerManager")

// peerManager is responsible for managing all the Peers, both online and offline
type peerManager struct {
	net  *Network
	stop chan interface{}
	Data chan PeerParcel

	//peerMutex  sync.RWMutex
	peers *PeerMap

	//onlinePeers map[string]bool // set of online peers
	incoming uint
	outgoing uint

	specialIP map[string]bool

	lastPeerRequest        time.Time
	lastPeerDial           time.Time
	lastPeerDuplicateCheck time.Time
	lastSeedRefresh        time.Time

	rng    *rand.Rand
	logger *log.Entry
}

// newpeerManager creates a new peer manager for the given controller
// configuration is shared between the two
func newpeerManager(network *Network) *peerManager {
	pm := &peerManager{}
	pm.net = network

	pm.logger = pmLogger.WithFields(log.Fields{
		"node":    pm.net.conf.NodeName,
		"port":    pm.net.conf.ListenPort,
		"network": pm.net.conf.Network})
	pm.logger.WithField("peermanager_init", pm.net.conf).Debugf("Initializing Peer Manager")

	pm.peers = NewPeerMap()

	pm.stop = make(chan interface{}, 1)
	pm.Data = make(chan PeerParcel, StandardChannelSize)

	// TODO parse config special peers
	pm.rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	return pm
}

// Start starts the peer manager
// reads from the seed and connect to peers
func (pm *peerManager) Start() {
	pm.logger.Info("Starting the Peer Manager")

	pm.bootStrapPeers()

	// TODO discover from seed
	// 		parse and dial special peers
	//go pm.receiveData()
	go pm.managePeers()
	go pm.manageData()
}

// Stop shuts down the peer manager and all active connections
func (pm *peerManager) Stop() {
	pm.stop <- true

	//pm.peerMutex.RLock()
	//defer pm.peerMutex.RUnlock()
	for _, p := range pm.peers.Slice() {
		p.GoOffline()
	}
}

func (pm *peerManager) bootStrapPeers() {
	// TODO load peers.json

	pm.lastSeedRefresh = time.Now()
	pm.discoverSeeds()
}

func (pm *peerManager) manageData() {
	for {
		data := <-pm.Data
		parcel := data.Parcel
		peer := data.Peer

		// wrong network
		if parcel.Header.Network != pm.net.conf.Network {
			pm.logger.Warnf("Peer %s tried to send a message for network %s, disconnecting", peer, parcel.Header.Network.String())
			pm.banPeer(peer)
			continue
		}

		if parcel.Header.NodeID == pm.net.conf.NodeID {
			pm.logger.Warnf("Peer %s is ourselves, banning", peer.ConnectAddress())
			pm.banPeer(peer)
			continue
		}

		switch parcel.Header.Type {
		case TypeMessagePart: // deprecated
		case TypeHeartbeat: // deprecated
		case TypePing:
		case TypePong:
		case TypeAlert:

		case TypeMessage: // Application message, send it on.
			ApplicationMessagesReceived++
			pm.net.FromNetwork.Send(parcel)
		case TypePeerRequest:
			if time.Since(peer.lastPeerSend) >= pm.net.conf.PeerRequestInterval {
				peer.lastPeerSend = time.Now()
				go pm.sharePeers(peer)
			} else {
				pm.logger.Warnf("Peer %s requested peer share sooner than expected", peer)
			}
		case TypePeerResponse:
			// TODO check here if we asked them for a peer request
			if time.Since(peer.lastPeerRequest) >= pm.net.conf.PeerRequestInterval {
				peer.lastPeerRequest = time.Now()
				go pm.processPeers(peer, parcel)
			} else {
				pm.logger.Warnf("Peer %s sent us an umprompted peer share", peer)
			}
		default:
			pm.logger.Warnf("Peer %s sent unknown parcel.Header.Type?: %+v ", peer, parcel)
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
			// TODO check if seed exists?
			seed := pm.SpawnPeer(address, true, port)
			pm.addPeer(seed)
		} else {
			pm.logger.Errorf("Bad peer in " + pm.net.conf.SeedURL + " [" + line + "]")
		}

	}

	pm.logger.Debugf("discoverSeed got peers: %s", report)
}

// processPeers processes a peer share response
func (pm *peerManager) processPeers(peer *Peer, parcel *Parcel) {
	var list []Peer
	err := json.Unmarshal(parcel.Payload, list)
	if err != nil {
		pm.logger.WithError(err).Warnf("Failed to unmarshal peer share from peer %s", peer)
	}

	known := make(map[string]bool)
	//pm.peerMutex.RLock()
	for _, p := range pm.peers.Slice() {
		known[p.ConnectAddress()] = true
	}
	//pm.peerMutex.RUnlock()

	for _, p := range list {
		if !known[p.ConnectAddress()] {
			known[p.ConnectAddress()] = true

			add := pm.SpawnPeer(p.Address, true, p.ListenPort)
			pm.addPeer(add)
		}
	}
}

// sharePeers creates a list of peers to share and sends it to peer
func (pm *peerManager) sharePeers(peer *Peer) {
	list := pm.filteredSharing()

	json, ok := json.Marshal(list)
	if ok != nil {
		pm.logger.WithError(ok).Error("Failed to marshal peer list to json")
		return
	}
	parcel := NewParcel(TypePeerResponse, json)
	peer.Send(parcel)
}

func (pm *peerManager) managePeers() {
	for {
		/*if time.Since(pm.lastPeerDuplicateCheck) > pm.net.conf.RedialInterval {
			pm.managePeersDetectDuplicate()
		}*/

		pm.logger.Debugf("Managing peers")

		if time.Since(pm.lastSeedRefresh) > pm.net.conf.PeerReseedInterval {
			pm.lastSeedRefresh = time.Now()
			pm.discoverSeeds()
		}

		// manage online connectivity
		if time.Since(pm.lastPeerDial) > pm.net.conf.RedialInterval {
			pm.lastPeerDial = time.Now()
			pm.managePeersDialOutgoing()
		}

		for _, p := range pm.peers.Slice() {
			fmt.Println("Checking peer", p)
			if p.IsOnline() {
				if time.Since(p.lastPeerRequest) > p.config.PeerRequestInterval {
					p.lastPeerRequest = time.Now()

					req := NewParcel(TypePeerRequest, []byte("Peer Request"))
					p.Send(req)
				}

				if time.Since(p.lastPeerSend) > pm.net.conf.PingInterval {
					// should update the lastpeersend on its own
					ping := NewParcel(TypePing, []byte("Ping"))
					p.Send(ping)
				}
			}
		}

		// manager peers every second
		time.Sleep(time.Second)
	}
}

func (pm *peerManager) managePeersDetectDuplicate() {
	exists := make(map[string]*Peer)
	var remove []*Peer
	//pm.peerMutex.RLock()
	for _, p := range pm.peers.Slice() {
		addr := p.ConnectAddress()
		if other, ok := exists[addr]; ok {
			if p.Better(other) {
				remove = append(remove, other)
				exists[addr] = p
			} else {
				remove = append(remove, p)
			}
		} else {
			exists[addr] = p
		}
	}
	//pm.peerMutex.RUnlock()

	if len(remove) > 0 {
		for _, p := range remove {
			pm.removePeer(p)
		}
	}
}

func (pm *peerManager) managePeersDialOutgoing() {
	var count uint // online OR dialing
	//pm.peerMutex.RLock()
	for _, p := range pm.peers.Slice() {
		if !p.IsOffline() {
			count++
			// TODO subtract special?
		}
	}
	//pm.peerMutex.RUnlock()

	pm.logger.Debugf("We have %d peers online or connecting", count)

	if want := int(pm.net.conf.Outgoing - count); want > 0 {
		filter := pm.filteredOutgoing()
		peers := pm.getOutgoingSelection(filter, want)
		for _, p := range peers {
			p.StartToDial()
		}
	}
}

func (pm *peerManager) SpawnPeer(address string, outgoing bool, listenPort string) *Peer {
	p := &Peer{Address: address, Outgoing: outgoing, state: Offline, ListenPort: listenPort}
	p.net = pm.net
	p.logger = peerLogger.WithFields(log.Fields{
		"node":       pm.net.conf.NodeName,
		"hash":       p.Hash,
		"address":    p.Address,
		"port":       p.Port,
		"listenPort": p.ListenPort,
		"outgoing":   p.Outgoing,
	})
	p.config = pm.net.conf
	p.ListenPort = p.config.ListenPort // assume they listen on same port we do
	p.stop = make(chan interface{}, 1)
	p.incoming = make(chan *Parcel, StandardChannelSize)
	p.Hash = address + ":" + listenPort // TODO make this a hash
	pm.logger.WithField("address", fmt.Sprintf("%s:%s", address, listenPort)).Debugf("Creating new peer %s", p)
	pm.addPeer(p)
	return p
}

// addPeer adds a peer to the manager system
func (pm *peerManager) addPeer(peer *Peer) {
	//pm.peerMutex.Lock()
	//defer pm.peerMutex.Unlock()

	pm.peers.Add(peer)

	if peer.Outgoing {
		pm.outgoing++
	} else {
		pm.incoming++
	}
}

func (pm *peerManager) banPeer(peer *Peer) {
	pm.removePeer(peer)
}

func (pm *peerManager) removePeer(peer *Peer) {
	//pm.peerMutex.Lock()
	//defer pm.peerMutex.Unlock()
	peer.GoOffline()
	pm.peers.Remove(peer)
}

func (pm *peerManager) HandleIncoming(con net.Conn) {
	ip := con.RemoteAddr().String()
	special := pm.specialIP[ip]

	ipLog := pm.logger.WithField("remote_addr", ip)

	if !special {
		if pm.outgoing >= pm.net.conf.Outgoing {
			ipLog.Info("Rejecting inbound connection because of inbound limit")
			con.Close()
			return
		} else if pm.net.conf.RefuseIncoming || pm.net.conf.RefuseUnknown {
			ipLog.WithFields(log.Fields{
				"RefuseIncoming": pm.net.conf.RefuseIncoming,
				"RefuseUnknown":  pm.net.conf.RefuseUnknown,
			}).Info("Rejecting inbound connection because of config settings")
			con.Close()
			return
		}
	}

	p := pm.SpawnPeer(ip, false, "0") // create AND ADD a peer
	p.StartWithActiveConnection(con)  // peer is online

	//c := NewConnection(con, pm.net.conf)

	// TODO check if special
	// TODO check if incoming is maxed out
	// TODO add peer
}

func (pm *peerManager) Broadcast(parcel *Parcel, full bool) {
	//pm.peerMutex.RLock()
	//defer pm.peerMutex.RUnlock()
	if full {
		for _, p := range pm.peers.Slice() {
			p.Send(parcel)
		}
		return
	}

	// fanout
	selection := pm.selectRandomPeers(pm.net.conf.Fanout)
	for _, p := range selection {
		p.Send(parcel)
	}
	// TODO always send to special
}

// filteredOutgoing generates a subset of peers that we can dial and
// are not already connected to
func (pm *peerManager) filteredOutgoing() []*Peer {
	var filtered []*Peer
	//pm.peerMutex.RLock()
	for _, p := range pm.peers.Slice() {
		if p.IsOffline() && p.CanDial() {
			filtered = append(filtered, p)
		}
	}
	//pm.peerMutex.RUnlock()

	return filtered
}

func (pm *peerManager) filteredSharing() []*Peer {
	var filtered []*Peer
	//pm.peerMutex.RLock()
	for _, p := range pm.peers.Slice() {
		if p.QualityScore >= pm.net.conf.MinimumQualityScore {
			filtered = append(filtered, p)
		}
	}
	//pm.peerMutex.RUnlock()

	return filtered
}

// getOutgoingSelection creates a subset of total connectable peers by getting
// as much prefix variation as possible
//
// Takes the input and spreads peers out over n equally sized buckets based on their
// ipv4 prefix, then iterates over those buckets and removes a random peer from each
// one until it has enough
func (pm *peerManager) getOutgoingSelection(filtered []*Peer, wanted int) []*Peer {
	// we have just enough
	if len(filtered) <= wanted {
		pm.logger.Debugf("getOutgoingSelection returning %d peers", len(filtered))
		return filtered
	}

	// generate a list of peers distant to each other
	buckets := make([][]*Peer, wanted)
	bucketSize := uint32(4294967295/uint32(wanted)) + 1 // 33554432 for wanted=128

	// distribute peers over n buckets
	for _, peer := range filtered {
		bucketIndex := int(peer.Location / bucketSize)
		buckets[bucketIndex] = append(buckets[bucketIndex], peer)
	}

	// pick random peers from each bucket
	var picked []*Peer
	for len(picked) < wanted {
		offset := pm.rng.Intn(len(buckets)) // start at a random point in the bucket array
		for i := 0; i < len(buckets); i++ {
			bi := (i + offset) % len(buckets)
			bucket := buckets[bi]
			if len(bucket) > 0 {
				pi := pm.rng.Intn(len(bucket)) // random member in bucket
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
	//pm.peerMutex.RLock()
	var peers []*Peer
	for _, p := range pm.peers.Slice() {
		if p.IsOnline() {
			peers = append(peers, p)
		}
	}
	//pm.peerMutex.RUnlock() // unlock early before a shuffle

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
		//pm.peerMutex.RLock()
		//defer pm.peerMutex.RUnlock()
		if peer := pm.peers.Get(hash); peer != nil {
			peer.Send(parcel)
		}
	}
}
