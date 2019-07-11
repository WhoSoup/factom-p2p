package p2p

import (
	"fmt"
	"net"
	"strconv"
)

type IP struct {
	Address string `json:"address"`
	Port    string `json:"port"`
}

// NewIP creates an IP struct from a given addr and port, throws error if addr could not be resolved
func NewIP(addr, port string) (IP, error) {
	if len(addr) == 0 || len(port) == 0 {
		return IP{}, fmt.Errorf("no address or port given (%s:%s)", addr, port)
	}

	parse := net.ParseIP(addr)
	if parse == nil {
		return IP{}, fmt.Errorf("unable to parse ip address: %s", addr)
	}

	ip := IP{addr, port}
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
