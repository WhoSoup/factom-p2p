package p2p

import (
	"fmt"
	"hash/crc32"
	"strconv"
)

type Handshake struct {
	Network      NetworkID
	Version      uint16
	Type         ParcelType
	NodeID       uint32
	ListenPort   string
	Loopback     uint64
	Alternatives []Endpoint
}

func (h *Handshake) Valid(conf *Configuration, loopback uint64) error {
	if h.Version < conf.ProtocolVersionMinimum {
		return fmt.Errorf("version %d is below the minimum", h.Version)
	}

	if h.Loopback == loopback {
		return fmt.Errorf("loopback")
	}

	if h.Network != conf.Network {
		return fmt.Errorf("wrong network id %x", h.Network)
	}

	port, err := strconv.Atoi(h.ListenPort)
	if err != nil {
		return fmt.Errorf("unable to parse port %s: %v", h.ListenPort, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("given port out of range: %d", port)
	}

	for _, ep := range h.Alternatives {
		if !ep.Valid() {
			return fmt.Errorf("invalid list of alternatives provided")
		}
	}

	return nil
}

func newHandshake(conf *Configuration, loopback uint64) *Handshake {
	hs := new(Handshake)
	hs.Type = TypeHandshake
	hs.Network = conf.Network
	hs.Version = conf.ProtocolVersion
	hs.NodeID = conf.NodeID
	hs.ListenPort = conf.ListenPort
	hs.Loopback = loopback
	return hs
}

// HandshakeGob is an alias of V9MSG for backward compatibility
type HandshakeGob V9Msg

// Valid checks if the other node is compatible
func (h *HandshakeGob) Valid(conf *Configuration) error {
	if h.Header.Version < conf.ProtocolVersionMinimum {
		return fmt.Errorf("version %d is below the minimum", h.Header.Version)
	}

	if h.Header.Network != conf.Network {
		return fmt.Errorf("wrong network id %x", h.Header.Network)
	}

	if len(h.Payload) == 0 {
		return fmt.Errorf("zero-length payload")
	}

	if h.Header.Length != uint32(len(h.Payload)) {
		return fmt.Errorf("length in header does not match payload")
	}

	csum := crc32.Checksum(h.Payload, crcTable)
	if csum != h.Header.Crc32 {
		return fmt.Errorf("invalid checksum")
	}

	port, err := strconv.Atoi(h.Header.PeerPort)
	if err != nil {
		return fmt.Errorf("unable to parse port %s: %v", h.Header.PeerPort, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("given port out of range: %d", port)
	}
	return nil
}

func newHandshakeGob(conf *Configuration, payload []byte) *HandshakeGob {
	hs := new(HandshakeGob)
	hs.Header = V9Header{
		Network:  conf.Network,
		Version:  conf.ProtocolVersion,
		Type:     TypeHandshake,
		NodeID:   uint64(conf.NodeID),
		PeerPort: conf.ListenPort,
		AppHash:  "NetworkMessage",
		AppType:  "Network",
	}
	hs.SetPayload(payload)
	return hs
}

// SetPayload adds a payload to the HandshakeGob and updates the header with metadata
func (h *HandshakeGob) SetPayload(payload []byte) {
	h.Payload = payload
	h.Header.Crc32 = crc32.Checksum(h.Payload, crcTable)
	h.Header.Length = uint32(len(h.Payload))
}

func newHandshakeV11(conf *Configuration, loopback []byte) *V11Handshake {
	hs := new(V11Handshake)
	hs.NetworkID = uint32(conf.Network)
	hs.Version = uint32(conf.ProtocolVersion)
	hs.ListenPort = conf.ListenPort
	hs.NodeID = conf.NodeID
	hs.Loopback = loopback
	return hs
}
