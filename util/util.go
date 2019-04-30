package util

import (
	"encoding/binary"
	"math/rand"
	"net"
)

func Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to Shuffle")
	}

	// Fisher-Yates shuffle: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// Shuffle really ought not be called with n that doesn't fit in 32 bits.
	// Not only will it take a very long time, but with 2³¹! possible permutations,
	// there's no way that any PRNG can have a big enough internal state to
	// generate even a minuscule percentage of the possible permutations.
	// Nevertheless, the right API signature accepts an int n, so handle it as best we can.
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(rand.Int63n(int64(i + 1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(rand.Int31n(int32(i + 1)))
		swap(i, j)
	}
}

// IP2Loc converts an ip address to a uint32
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
