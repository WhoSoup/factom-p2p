package p2p

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

// Persist is the object that gets json-marshalled and written to disk
type Persist struct {
	Bans      map[string]time.Time `json:"bans"`
	Bootstrap []IP                 `json:"bootstrap"`
}

// wrappers for reading and writing the peer file
func (c *controller) writePersistFile(data []byte) error {
	if c.net.conf.PersistFile == "" {
		return nil
	}
	return ioutil.WriteFile(c.net.conf.PersistFile, data, 0644) // rw r r
}

func (c *controller) loadPersistFile() ([]byte, error) {
	if c.net.conf.PersistFile == "" {
		return nil, nil
	}
	return ioutil.ReadFile(c.net.conf.PersistFile)
}

func (c *controller) PersistData() ([]byte, error) {
	var pers Persist
	pers.Bans = make(map[string]time.Time)

	c.banMtx.Lock()
	now := time.Now()
	for ip, end := range c.Bans {
		if end.Before(now) {
			delete(c.Bans, ip)
		} else {
			pers.Bans[ip] = end
		}
	}
	c.banMtx.Unlock()

	peers := c.peers.Slice()
	pers.Bootstrap = make([]IP, len(peers))
	for i, p := range peers {
		pers.Bootstrap[i] = p.IP
	}

	return json.Marshal(pers)
}

func (c *controller) ParsePersist(data []byte) (*Persist, error) {
	var pers Persist
	err := json.Unmarshal(data, &pers)
	if err != nil {
		return nil, err
	}

	// decoding from a blank or invalid file
	if pers.Bans == nil {
		pers.Bans = make(map[string]time.Time)
	}

	c.logger.Debugf("bootstrapping with %d ips and %d bans", len(pers.Bootstrap), len(pers.Bans))
	return &pers, nil
}
