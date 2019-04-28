package p2p

import (
	"fmt"
	"hash/crc32"

	log "github.com/sirupsen/logrus"
)

var (
	crcTable     = crc32.MakeTable(crc32.Koopman)
	parcelLogger = packageLogger.WithField("subpack", "connection")
)

// Parcel is the atomic level of communication for the p2p network.  It contains within it the necessary info for
// the networking protocol, plus the message that the Application is sending.
type Parcel struct {
	Header  ParcelHeader
	Payload []byte
}

// ParcelHeaderSize is the number of bytes in a parcel header
const ParcelHeaderSize = 32

type ParcelHeader struct {
	Network     NetworkID         // 4 bytes - the network we are on (eg testnet, main net, etc.)
	Version     uint16            // 2 bytes - the version of the protocol we are running.
	Type        ParcelCommandType // 2 bytes - network level commands (eg: ping/pong)
	Length      uint32            // 4 bytes - length of the payload (that follows this header) in bytes
	TargetPeer  string            // ? bytes - "" or nil for broadcast, otherwise the destination peer's hash.
	Crc32       uint32            // 4 bytes - data integrity hash (of the payload itself.)
	Part        uint16            // 2 bytes - in case of multipart parcels, indicates which part this corresponds to, otherwise should be 0
	Parts       uint16            // 2 bytes - in case of multipart parcels, indicates the total number of parts that the receiver should expect
	NodeID      uint64
	PeerAddress string // address of the peer set by connection to know who sent message (for tracking source of other peers)
	PeerPort    string // port of the peer , or we are listening on
	AppHash     string // Application specific message hash, for tracing
	AppType     string // Application specific message type, for tracing
}

type ParcelCommandType uint16

// Parcel commands -- all new commands should be added to the *end* of the list!
const ( // iota is reset to 0
	TypeHeartbeat    ParcelCommandType = iota // "Note, I'm still alive"
	TypePing                                  // "Are you there?"
	TypePong                                  // "yes, I'm here"
	TypePeerRequest                           // "Please share some peers"
	TypePeerResponse                          // "Here's some peers I know about."
	TypeAlert                                 // network wide alerts (used in bitcoin to indicate criticalities)
	TypeMessage                               // Application level message
	TypeMessagePart                           // Application level message that was split into multiple parts
)

// CommandStrings is a Map of command ids to strings for easy printing of network comands
var CommandStrings = map[ParcelCommandType]string{
	TypeHeartbeat:    "Heartbeat",     // "Note, I'm still alive"
	TypePing:         "Ping",          // "Are you there?"
	TypePong:         "Pong",          // "yes, I'm here"
	TypePeerRequest:  "Peer-Request",  // "Please share some peers"
	TypePeerResponse: "Peer-Response", // "Here's some peers I know about."
	TypeAlert:        "Alert",         // network wide alerts (used in bitcoin to indicate criticalities)
	TypeMessage:      "Message",       // Application level message
	TypeMessagePart:  "MessagePart",   // Application level message that was split into multiple parts
}

func (p *Parcel) LogEntry() *log.Entry {
	return parcelLogger.WithFields(log.Fields{
		"network":     p.Header.Network.String(),
		"version":     p.Header.Version,
		"app_hash":    p.Header.AppHash,
		"app_type":    p.Header.AppType,
		"command":     CommandStrings[p.Header.Type],
		"length":      p.Header.Length,
		"target_peer": p.Header.TargetPeer,
		"crc32":       p.Header.Crc32,
		//"node_id":     p.Header.NodeID,
	})
}

func (p *Parcel) MessageType() string {
	return fmt.Sprintf("[%s]", CommandStrings[p.Header.Type])
}

func (p *Parcel) String() string {
	return fmt.Sprintf("[%s] %dB v%d", CommandStrings[p.Header.Type], p.Header.Length, p.Header.Version)
}

func NewMessage(payload []byte) *Parcel {
	return NewParcel(TypeMessage, payload)
}

func NewParcel(command ParcelCommandType, payload []byte) *Parcel {
	parcel := new(Parcel)
	parcel.Header = ParcelHeader{ // the header information will get filled more when sending
		Type:    command,
		AppHash: "NetworkMessage",
		AppType: "Network"}
	parcel.SetPayload(payload)
	return parcel
}

func (p *Parcel) SetPayload(payload []byte) {
	p.Payload = payload
	p.Header.Crc32 = crc32.Checksum(p.Payload, crcTable)
	p.Header.Length = uint32(len(p.Payload))
}

func (p *Parcel) Valid() error {
	if p == nil {
		return fmt.Errorf("nil parcel")
	}

	head := p.Header
	if head.Version == 0 {
		return fmt.Errorf("invalid version")
	}

	if head.Type >= ParcelCommandType(len(CommandStrings)) {
		return fmt.Errorf("unknown parcel type %d", head.Type)
	}

	if head.Length != uint32(len(p.Payload)) {
		return fmt.Errorf("length in header does not match payload")
	}

	csum := crc32.Checksum(p.Payload, crcTable)
	if csum != head.Crc32 {
		return fmt.Errorf("invalid checksum")
	}

	return nil
}
