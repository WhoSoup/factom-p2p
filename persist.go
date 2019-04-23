package p2p

import "time"

type Persist struct {
	IPs  []IP         `json:"ips"`
	Bans []PersistBan `json:"bans"`
}

type PersistBan struct {
	Address string    `json:"address"`
	Time    time.Time `json:"time"`
}
