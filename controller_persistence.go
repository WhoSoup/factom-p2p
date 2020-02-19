package p2p

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

// PeerCache is the object that gets json-marshalled and written to disk
type PeerCache struct {
	Bans  map[string]time.Time `json:"bans"` // can be ip or ip:port
	Peers []Endpoint           `json:"peers"`
}

func (c *controller) loadPeerCache() (*PeerCache, error) {
	cache, err := c.loadPeerCacheFile()
	if err != nil {
		return nil, err
	}

	if len(cache) > 0 {
		return c.parsePeerCache(cache)
	}

	return nil, nil
}

// wrappers for reading and writing the peer file
func (c *controller) writePeerCacheFile(data []byte) error {
	if c.net.conf.PeerCacheFile == "" {
		return nil
	}
	return ioutil.WriteFile(c.net.conf.PeerCacheFile, data, 0644) // rw r r
}

func (c *controller) loadPeerCacheFile() ([]byte, error) {
	if c.net.conf.PeerCacheFile == "" {
		return nil, nil
	}
	return ioutil.ReadFile(c.net.conf.PeerCacheFile)
}

func (c *controller) peerCacheData() ([]byte, error) {
	var pers PeerCache
	pers.Bans = make(map[string]time.Time)

	c.banMtx.Lock()
	now := time.Now()
	for addr, end := range c.bans {
		if end.Before(now) {
			delete(c.bans, addr)
		} else {
			pers.Bans[addr] = end
		}
	}
	c.banMtx.Unlock()

	peers := c.peers.Slice()
	pers.Peers = make([]Endpoint, len(peers))
	for i, p := range peers {
		pers.Peers[i] = p.Endpoint
	}

	return json.Marshal(pers)
}

func (c *controller) parsePeerCache(data []byte) (*PeerCache, error) {
	var pers PeerCache
	err := json.Unmarshal(data, &pers)
	if err != nil {
		return nil, err
	}

	// decoding from a blank or invalid file
	if pers.Bans == nil {
		pers.Bans = make(map[string]time.Time)
	}

	c.logger.Debugf("bootstrapping with %d ips and %d bans", len(pers.Peers), len(pers.Bans))
	return &pers, nil
}

func (c *controller) persistPeerFile() {
	if c.net.conf.PeerCacheFile == "" {
		return
	}

	data, err := c.peerCacheData()
	if err != nil {
		c.logger.WithError(err).Warn("unable to create peer cache data")
	} else {
		err = c.writePeerCacheFile(data)
		if err != nil {
			c.logger.WithError(err).Warn("unable to persist peer cache data")
		}
	}
}
