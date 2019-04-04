// Copyright 2017 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"fmt"
	"hash/crc32"
	"math/rand"

	"github.com/FactomProject/factomd/common/primitives"
)

var prLogger = packageLogger.WithField("subpack", "protocol")

// This file contains the global variables and utility functions for the p2p network operation.  The global variables and constants can be tweaked here.

func BlockFreeParcelSend(channel chan *Parcel, parcel *Parcel) int {
	removed := 0
	highWaterMark := int(float64(cap(channel)) * 0.95)
	clen := len(channel)
	switch {
	case highWaterMark < clen:
		prLogger.Warnf("BlockFreeParcelSend() - DROPPING MESSAGES. Channel is over 90 percent full! \n channel len: \n %d \n 90 percent: \n %d \n last message type: %v", len(channel), highWaterMark, parcel.Header.Type)
		for highWaterMark <= len(channel) { // Clear out some messages
			removed++
			<-channel
		}
		fallthrough
	default:
		select { // hits default if sending message would block.
		case channel <- parcel:
		default:
		}
	}
	return removed
}

// BlockFreeChannelSend will remove things from the queue to make room for new messages if the queue is full.
// This prevents channel blocking on full.
//		Returns: The number of elements cleared from the channel to make room
func BlockFreeChannelSend(channel chan interface{}, message interface{}) int {
	removed := 0
	highWaterMark := int(float64(cap(channel)) * 0.95)
	clen := len(channel)
	switch {
	case highWaterMark < clen:
		str, _ := primitives.EncodeJSONString(message)
		prLogger.Warnf("nonBlockingChanSend() - DROPPING MESSAGES. Channel is over 90 percent full! \n channel len: \n %d \n 90 percent: \n %d \n last message type: %v", len(channel), highWaterMark, str)
		for highWaterMark <= len(channel) { // Clear out some messages
			removed++
			<-channel
		}
		fallthrough
	default:
		select { // hits default if sending message would block.
		case channel <- message:
		default:
		}
	}
	return removed
}

// Global variables for the p2p protocol
var (
	StandardChannelSize = 5000
	BroadcastFlag       = "<BROADCAST>"
	FullBroadcastFlag   = "<FULLBORADCAST>"
	RandomPeerFlag      = "<RANDOMPEER>"
	// Testing metrics
	TotalMessagesReceived       uint64
	TotalMessagesSent           uint64
	ApplicationMessagesReceived uint64

	CRCKoopmanTable = crc32.MakeTable(crc32.Koopman)
	RandomGenerator *rand.Rand // seeded pseudo-random number generator

)

const (
	// ProtocolVersion is the latest version this package supports
	ProtocolVersion uint16 = 9
	// ProtocolVersionMinimum is the earliest version this package supports
	ProtocolVersionMinimum uint16 = 9
)

// NetworkIdentifier represents the P2P network we are participating in (eg: test, nmain, etc.)
type NetworkID uint32

// Network indicators.
const (
	// MainNet represents the production network
	MainNet NetworkID = 0xfeedbeef

	// TestNet represents a testing network
	TestNet NetworkID = 0xdeadbeef

	// LocalNet represents any arbitrary/private network
	LocalNet NetworkID = 0xbeaded
)

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
