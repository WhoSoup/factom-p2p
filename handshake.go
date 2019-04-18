package p2p

import (
	"fmt"
	"strconv"
)

type Handshake struct {
	NodeID  uint64
	Port    string
	Version uint16
	Network NetworkID
}

func (h Handshake) Verify(id uint64, min uint16, n NetworkID) error {
	if h.NodeID == id {
		return fmt.Errorf("connected to ourselves")
	}

	if h.Version < min {
		return fmt.Errorf("version %d is below the minimum", h.Version)
	}

	if h.Network != n {
		return fmt.Errorf("wrong network id %x", h.Network)
	}

	port, err := strconv.Atoi(h.Port)
	if err != nil {
		return fmt.Errorf("unable to parse port %s: %v", h.Port, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("given port out of range: %d", port)
	}

	return nil
}
