package p2p

import (
	"encoding/binary"
	"fmt"

	"github.com/FactomProject/factomd/common/primitives"
)

// NetworkIdentifier represents the P2P network we are participating in (eg: test, nmain, etc.)
type NetworkID uint32

// NetworkID are specific uint32s to identify separate networks
//
// The default identifiers are MainNet (the main production network), TestNet (for network=TESTNET)
// and LocalNet (for network=LOCAL).
//
// Custom NetworkIDs (network=CUSTOM) are generated from the "customnet" command line flag
const (
	MainNet  NetworkID = 0xfeedbeef
	TestNet  NetworkID = 0xdeadbeef
	LocalNet NetworkID = 0xbeaded
)

// NewNetworkID converts a string to a network id
func NewNetworkID(name string) NetworkID {
	hashSuffix := primitives.Sha([]byte(name)).Bytes()[:4]
	return NetworkID(binary.BigEndian.Uint32(hashSuffix))
}

func (n *NetworkID) String() string {
	switch *n {
	case MainNet:
		return "MainNet"
	case TestNet:
		return "TestNet"
	case LocalNet:
		return "LocalNet"
	default:
		return fmt.Sprintf("CustomNet ID: %x\n", *n)
	}
}
