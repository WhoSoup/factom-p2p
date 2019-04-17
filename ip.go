package p2p

import "fmt"

type IP struct {
	Address  string
	Port     string
	Location uint32
}

func NewIP(addr, port string) (IP, error) {
	ip := IP{addr, port, 0}
	loc, err := IP2Location(addr)
	if err != nil {
		return ip, err
	}
	ip.Location = loc
	return ip, nil
}

func (ip IP) String() string {
	return fmt.Sprintf("%s:%s", ip.Address, ip.Port)
}
