package p2p

import (
	"fmt"
	"hash/crc32"
	"strconv"
)

var (
	crcTable = crc32.MakeTable(crc32.Koopman)
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
	Network     NetworkID  // 4 bytes - the network we are on (eg testnet, main net, etc.)
	Version     uint16     // 2 bytes - the version of the protocol we are running.
	Type        ParcelType // 2 bytes - network level commands (eg: ping/pong)
	Length      uint32     // 4 bytes - length of the payload (that follows this header) in bytes
	TargetPeer  string     // ? bytes - "" or nil for broadcast, otherwise the destination peer's hash.
	Crc32       uint32     // 4 bytes - data integrity hash (of the payload itself.)
	Part        uint16     // 2 bytes - in case of multipart parcels, indicates which part this corresponds to, otherwise should be 0
	Parts       uint16     // 2 bytes - in case of multipart parcels, indicates the total number of parts that the receiver should expect
	NodeID      uint64
	PeerAddress string // address of the peer set by connection to know who sent message (for tracking source of other peers)
	PeerPort    string // port of the peer , or we are listening on
	AppHash     string // Application specific message hash, for tracing
	AppType     string // Application specific message type, for tracing
}

func (h *ParcelHeader) Valid(conf *Configuration) error {
	if h.NodeID == conf.NodeID {
		return (fmt.Errorf("connected to ourselves"))
	}

	if h.Version < conf.ProtocolVersionMinimum {
		return (fmt.Errorf("version %d is below the minimum", h.Version))
	}

	if h.Network != conf.Network {
		return (fmt.Errorf("wrong network id %x", h.Network))
	}

	port, err := strconv.Atoi(h.PeerPort)
	if err != nil {
		return (fmt.Errorf("unable to parse port %s: %v", h.PeerPort, err))
	}

	if port < 1 || port > 65535 {
		return (fmt.Errorf("given port out of range: %d", port))
	}
	return nil
}

func (p *Parcel) IsApplicationMessage() bool {
	switch p.Header.Type {
	case TypeMessage, TypeMessagePart:
		return true
	default:
		return false
	}
}
func (p *Parcel) SetAppData(appType, appHash string) {
	p.Header.AppHash = appHash
	p.Header.AppType = appType
}

func (p *Parcel) String() string {
	return fmt.Sprintf("[%s] %dB v%d", p.Header.Type, p.Header.Length, p.Header.Version)
}

func NewMessage(payload []byte) *Parcel {
	return newParcel(TypeMessage, payload)
}

func newParcel(command ParcelType, payload []byte) *Parcel {
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

func (p *Parcel) setMeta(conf *Configuration) {
	p.Header.NodeID = conf.NodeID
	p.Header.Version = conf.ProtocolVersion
	p.Header.Network = conf.Network
	p.Header.PeerPort = conf.ListenPort
}

// Valid checks header for inconsistencies
func (p *Parcel) Valid() error {
	if p == nil {
		return fmt.Errorf("nil parcel")
	}

	head := p.Header
	if head.Version == 0 {
		return fmt.Errorf("invalid version")
	}

	if head.Type >= ParcelType(len(typeStrings)) {
		return fmt.Errorf("unknown parcel type %d", head.Type)
	}

	if len(p.Payload) == 0 {
		return fmt.Errorf("zero-length payload")
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
