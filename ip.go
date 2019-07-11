package p2p

import (
	"fmt"
	"net"
	"strconv"
)

type IP struct {
	Address  string `json:"address"`
	Port     string `json:"port"`
	Location uint32 `json:"location,omitempty"`
}

// NewIP creates an IP struct from a given addr and port, throws error if addr could not be resolved
func NewIP(addr, port string) (IP, error) {
	if len(addr) == 0 || len(port) == 0 {
		return IP{}, fmt.Errorf("no address or port given (%s:%s)", addr, port)
	}
	ip := IP{addr, port, 0}
	loc, err := IP2Location(addr)
	if err != nil {
		return IP{}, err
	}
	ip.Location = loc
	return ip, nil
}

// ParseAddress takes input in the form of "address:port" and returns its IP
func ParseAddress(s string) (IP, error) {
	ip, port, err := net.SplitHostPort(s)
	if err != nil {
		return IP{}, err
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return IP{}, err
	}

	if p < 1 || p > 65535 {
		return IP{}, fmt.Errorf("port out of range")
	}

	return NewIP(ip, port)
}

func (ip IP) String() string {
	return fmt.Sprintf("%s:%s", ip.Address, ip.Port)
}
