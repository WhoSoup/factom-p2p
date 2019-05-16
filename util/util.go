package util

import (
	"crypto/sha256"
	"encoding/binary"
	"net"
)

// IP2Location converts an ip address to a uint32
//
// If the address is a hostmask, it attempts to resolve the address first
func IP2Location(addr string) (uint32, error) {
	// Split the IPv4 octets
	ip := net.ParseIP(addr)
	if ip == nil {
		ipAddress, err := net.LookupHost(addr)
		if err != nil {
			return 0, err // We use location on 0 to say invalid
		}
		addr = ipAddress[0]
		ip = net.ParseIP(addr)
	}
	if len(ip) == 16 { // If we got back an IP6 (16 byte) address, use the last 4 byte
		ip = ip[12:]
	}

	return binary.BigEndian.Uint32(ip), nil
}

// StringToUint32 hashes the input to generate a deterministic number representation
func StringToUint32(input string) uint32 {
	hash := sha256.Sum256([]byte(input))
	return binary.BigEndian.Uint32(hash[:4])
}
