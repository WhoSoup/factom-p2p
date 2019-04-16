package p2p

import (
	"bufio"
	"encoding/gob"
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
	net            *Network
	stop           chan interface{}
	peerDisconnect chan *Peer

	//	tempPeers *PeerList
	peers     *PeerList
	endpoints *EndpointMap

	lastPeerDial    time.Time
	lastSeedRefresh time.Time

	logger *log.Entry
}

// newPeerManager creates a new peer manager for the given controller
// configuration is shared between the two
func newPeerManager(network *Network) *peerManager {
	pm := &peerManager{}
	pm.net = network

	pm.logger = pmLogger.WithFields(log.Fields{
		"node":    pm.net.conf.NodeName,
		"port":    pm.net.conf.ListenPort,
		"network": pm.net.conf.Network})
	pm.logger.WithField("peermanager_init", pm.net.conf).Debugf("Initializing Peer Manager")

	pm.peers = NewPeerList()
	pm.endpoints = NewEndpointMap(pm.net)
	//pm.tempPeers = NewPeerList()

	pm.stop = make(chan interface{}, 1)
	pm.peerDisconnect = make(chan *Peer, 10) // TODO reconsider this value

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
}

// Stop shuts down the peer manager and all active connections
func (pm *peerManager) Stop() {
	pm.stop <- true

	for _, p := range pm.peers.Slice() {
		p.Stop()
	}
}

func (pm *peerManager) bootStrapPeers() {
	// TODO load peers.json
	pm.lastSeedRefresh = time.Now()
	pm.discoverSeeds()
}

func (pm *peerManager) manageData() {
	for {
		data := <-pm.net.peerParcel
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
			if ep := pm.endpoints.Get(address, port); ep != nil { // check if seed exists already
				ep.Seed = true
				continue
			}

			if ep, err := pm.endpoints.Create(address, port); err != nil {
				pm.logger.WithError(err).Debugf("Invalid endpoint in seed list: %s", line)
			} else {
				ep.Seed = true
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
		_, err := pm.endpoints.Create(p.Address, p.Port)
		if err != nil {
			pm.logger.WithError(err).Infof("Unable to register endpoint %s:%s from peer %s", p.Address, p.Port, peer)
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

func (pm *peerManager) managePeers() {
	for {
		/*if time.Since(pm.lastPeerDuplicateCheck) > pm.net.conf.RedialInterval {
			pm.managePeersDetectDuplicate()
		}*/

		//pm.logger.Debugf("Managing peers")

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
				pm.logger.Debugf("Pinging %s", p.Hash)
				ping := NewParcel(TypePing, []byte("Ping"))
				p.Send(ping)
			}
		}

		// manager peers every second
		time.Sleep(time.Second)
	}
}

func (pm *peerManager) managePeersDialOutgoing() {
	//pm.logger.Debugf("We have %d peers online or connecting", pm.endpoints.Online)

	// TODO rewrite to endpoints

	/*count := uint(pm.peers.Len())
	if want := int(pm.net.conf.Outgoing - count); want > 0 {
		filter := pm.filteredOutgoing()

		if len(filter) > 0 {
			peers := pm.getOutgoingSelection(filter, want)
			for _, p := range peers {
				if !pm.endpoints.Get() {
					p.StartToDial()
				}
			}
		}
	}*/
}

func (pm *peerManager) HandleIncoming(con net.Conn) {
	addr, _, err := net.SplitHostPort(con.RemoteAddr().String())
	if err != nil {
		pm.logger.WithError(err).Debugf("Unable to parse address %s", con.RemoteAddr().String())
		con.Close()
		return
	}

	// TODO allow special peers
	if uint(pm.peers.Len()) >= pm.net.conf.Incoming {
		pm.logger.Infof("Refusing incoming connection from %s because we are maxed out", con.RemoteAddr().String())
		con.Close()
		return
	}

	endpoint, err := pm.endpoints.Create(addr, "")
	if err != nil {
		pm.logger.WithError(err).Debugf("Unable to register endpoint %s", addr)
	}

	// TODO limit by endpoint

	go pm.processHandshake(endpoint, con, true)
}

func (pm *peerManager) Dial(addr, port string) {
	pm.logger.Debugf("Dialing to %s:%s", addr, port)

	endpoint, err := pm.endpoints.Create(addr, port)
	if err != nil {
		pm.logger.WithError(err).Errorf("Unable to create endpoint %s:%s", addr, port)
	}

	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", pm.net.conf.BindIP))
	if err != nil {
		pm.logger.WithError(err).Errorf("Unable to resolve local interface \"%s:0\"", pm.net.conf.BindIP)
		return
	}

	dialer := net.Dialer{
		LocalAddr: local,
		Timeout:   pm.net.conf.DialTimeout,
	}
	con, err := dialer.Dial("tcp", fmt.Sprintf("%s:%s", addr, port))

	if err != nil {
		pm.logger.WithError(err).Infof("Unable to reach peer %s:%s", addr, port)
		return
	}

	go pm.processHandshake(endpoint, con, false)
}

func (pm *peerManager) processHandshake(endpoint *Endpoint, con net.Conn, incoming bool) {
	addr := endpoint.Address
	tmplogger := pm.logger.WithField("addr", addr)
	timeout := time.Now().Add(pm.net.conf.HandshakeTimeout)
	handshake := Handshake{
		NodeID:  pm.net.conf.NodeID,
		Port:    pm.net.conf.ListenPort,
		Version: pm.net.conf.ProtocolVersion}
	encoder := gob.NewEncoder(con)
	con.SetWriteDeadline(timeout)
	err := encoder.Encode(handshake)

	if err != nil {
		tmplogger.WithError(err).Debugf("Failed to send handshake to incoming connection")
		con.Close()
		return
	}

	decoder := gob.NewDecoder(con)
	con.SetReadDeadline(timeout)
	err = decoder.Decode(&handshake)
	if err != nil {
		tmplogger.WithError(err).Debugf("Failed to read handshake from incoming connection")
		con.Close()
		return
	}

	err = handshake.Verify(pm.net.conf.NodeID, pm.net.conf.ProtocolVersionMinimum, pm.net.conf.Network)
	if err != nil {
		tmplogger.WithError(err).Debug("Handshake failed")
		con.Close()
		return
	}

	peer := NewPeer(pm.net, addr, handshake, pm.peerDisconnect)
	pm.peers.Add(peer)
	peer.StartWithConnection(con, incoming)
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
func (pm *peerManager) filteredOutgoing() []*Peer {
	// TODO rework with endpoint
	return nil
	/*	var filtered []*Peer
		for _, p := range pm.peers.Slice() {
			if !p.IsIncoming && p.IsOffline() && p.CanDial() {
				filtered = append(filtered, p)
			}
		}

		return filtered*/
}

func (pm *peerManager) filteredSharing(peer *Peer) []PeerShare {
	var filtered []PeerShare
	return filtered
	// TODO rework with endpoint
	/*
		shared := make(map[string]bool)
		for _, p := range pm.peers.Slice() {
			if !shared[p.ConnectAddress()] && p.Shareable() && p.Hash != peer.Hash {
				shared[p.ConnectAddress()] = true
				filtered = append(filtered, p.PeerShare())
			}
		}
		return filtered*/
}

// getOutgoingSelection creates a subset of total connectable peers by getting
// as much prefix variation as possible
//
// Takes the input and spreads peers out over n equally sized buckets based on their
// ipv4 prefix, then iterates over those buckets and removes a random peer from each
// one until it has enough
func (pm *peerManager) getOutgoingSelection(filtered []*Peer, wanted int) []*Peer {
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
		return []*Peer{filtered[rand]}
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
	var peers []*Peer
	for _, p := range pm.peers.Slice() {
		peers = append(peers, p)
	}

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
		if peer := pm.peers.Find(hash); peer != nil {
			peer.Send(parcel)
		}
	}
}
