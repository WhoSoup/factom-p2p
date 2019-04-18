package p2p

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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

	//	tempPeers *PeerList
	peers     *PeerStore
	endpoints *EndpointMap
	dialer    *Dialer

	lastPeerDial    time.Time
	lastSeedRefresh time.Time

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

	pm.peers = NewPeerStore()
	pm.endpoints = NewEndpointMap(pm.net)
	//pm.tempPeers = NewPeerList()
	pm.dialer = NewDialer(c.BindIP, c.RedialInterval, c.DialTimeout, c.RedialAttempts)

	pm.peerDisconnect = make(chan *Peer, 10) // TODO reconsider this value

	pm.stopPeers = make(chan bool, 1)
	pm.stopData = make(chan bool, 1)
	pm.stopOnline = make(chan bool, 1)

	// TODO parse config special peers
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
	// TODO load peers.json
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
				fmt.Println("test")
				ApplicationMessagesReceived++
				pm.net.FromNetwork.Send(parcel)
			case TypePeerRequest:
				go pm.sharePeers(peer)
				/*			if time.Since(peer.lastPeerSend) >= pm.net.conf.PeerRequestInterval {
								peer.lastPeerSend = time.Now()

							} else {
								pm.logger.Warnf("Peer %s requested peer share sooner than expected", peer)
							}*/
			case TypePeerResponse:
				go pm.processPeers(peer, parcel)
				/*			// TODO check here if we asked them for a peer request
							if time.Since(peer.lastPeerRequest) >= pm.net.conf.PeerRequestInterval {
								peer.lastPeerRequest = time.Now()

							} else {
								pm.logger.Warnf("Peer %s sent us an umprompted peer share", peer)
							}*/
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
				pm.endpoints.Register(ip, false)
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
			pm.endpoints.Register(ip, false)
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
	for _, ip := range pm.endpoints.IPs {
		if ip.Address != peer.Address || ip.Port != peer.Port {
			filtered = append(filtered, PeerShare{
				Address:      ip.Address,
				Port:         ip.Port,
				QualityScore: 1,
			})
		}
	}
	return filtered
}
func (pm *peerManager) managePeers() {
	pm.logger.Debug("Start managePeers()")
	defer pm.logger.Debug("Stop managePeers()")

	for {
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
			if time.Since(p.lastPeerRequest) > pm.net.conf.PeerRequestInterval {
				p.lastPeerRequest = time.Now()

				pm.logger.Debugf("Requesting peers from %s", p.Hash)
				req := NewParcel(TypePeerRequest, []byte("Peer Request"))
				p.Send(req)
			}

			if time.Since(p.LastSend) > pm.net.conf.PingInterval {
				ping := NewParcel(TypePing, []byte("Ping"))
				p.Send(ping)
			}
		}

		// manager peers every second

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
		filter := pm.filteredOutgoing()

		if len(filter) > 0 {
			peers := pm.getOutgoingSelection(filter, want)
			for _, p := range peers {
				if !pm.peers.IsConnected(p.Address) {
					go pm.Dial(p)
				}
			}
		}
	}
}

func (pm *peerManager) allowConnection(addr string) error {
	// TODO allow special peers

	if pm.net.conf.RefuseIncoming {
		return fmt.Errorf("Refusing all incoming connections")
	}

	if uint(pm.peers.Total()) >= pm.net.conf.Incoming {
		return fmt.Errorf("Refusing incoming connection from %s because we are maxed out", addr)
	}

	if pm.net.conf.PeerIPLimit > 0 && uint(pm.peers.Count(addr)) >= pm.net.conf.PeerIPLimit {
		return fmt.Errorf("Rejecting %s due to per ip limit of %d", addr, pm.net.conf.PeerIPLimit)
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

	if err = pm.allowConnection(addr); err != nil {
		pm.logger.WithError(err).Infof("Rejecting connection")
		con.Close()
		return
	}

	// TODO limit by endpoint
	peer := NewPeer(pm.net, pm.peerDisconnect)
	if peer.StartWithHandshake(ip, con, true) {
		old := pm.peers.Replace(peer)

		if ip.Port == "" {
			ip.Port = peer.Port
		}

		pm.endpoints.Register(ip, false)
		if old != nil {
			old.Stop(false)
		}
		pm.dialer.Reset(ip)
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
		if ip.Port == "" {
			ip.Port = peer.Port
		}
		if old := pm.peers.Replace(peer); old != nil {
			old.Stop(false)
		}

		pm.endpoints.Register(ip, false)
		pm.dialer.Reset(ip)
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

// filteredOutgoing generates a subset of peers that we can dial and
// are not already connected to
func (pm *peerManager) filteredOutgoing() []IP {
	var filtered []IP
	for _, p := range pm.endpoints.IPs {
		if pm.dialer.CanDial(p) && !pm.endpoints.IsIncoming(p) {
			filtered = append(filtered, p)
		}
	}

	return filtered
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
