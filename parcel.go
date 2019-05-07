package p2p

import (
	"fmt"
	"hash/crc32"
)

var (
	crcTable = crc32.MakeTable(crc32.Koopman)
)

// Parcel is the atomic level of communication for the p2p network.  It contains within it the necessary info for
// the networking protocol, plus the message that the Application is sending.
type Parcel struct {
	Type    ParcelType // 2 bytes - network level commands (eg: ping/pong)
	Address string     // ? bytes - "" or nil for broadcast, otherwise the destination peer's hash.
	Payload []byte
}

func (p *Parcel) IsApplicationMessage() bool {
	switch p.Type {
	case TypeMessage, TypeMessagePart:
		return true
	default:
		return false
	}
}

func (p *Parcel) String() string {
	return fmt.Sprintf("[%s] %dB", p.Type, len(p.Payload))
}

func NewMessage(payload []byte) *Parcel {
	return newParcel(TypeMessage, payload)
}

func newParcel(command ParcelType, payload []byte) *Parcel {
	parcel := new(Parcel)
	parcel.Type = command
	parcel.Payload = payload
	return parcel
}

// Valid checks header for inconsistencies
func (p *Parcel) Valid() error {
	if p == nil {
		return fmt.Errorf("nil parcel")
	}

	if p.Type >= ParcelType(len(typeStrings)) {
		return fmt.Errorf("unknown parcel type %d", p.Type)
	}

	if len(p.Payload) == 0 {
		return fmt.Errorf("zero-length payload")
	}

	return nil
}
