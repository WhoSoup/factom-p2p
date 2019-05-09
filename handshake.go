package p2p

import (
	"fmt"
	"hash/crc32"
	"strconv"
)

type Handshake V9Msg

func (h *Handshake) Valid(conf *Configuration) error {
	if h.Header.NodeID == uint64(conf.NodeID) {
		return (fmt.Errorf("connected to ourselves"))
	}

	if h.Header.Version < conf.ProtocolVersionMinimum {
		return (fmt.Errorf("version %d is below the minimum", h.Header.Version))
	}

	if h.Header.Network != conf.Network {
		return (fmt.Errorf("wrong network id %x", h.Header.Network))
	}

	port, err := strconv.Atoi(h.Header.PeerPort)
	if err != nil {
		return (fmt.Errorf("unable to parse port %s: %v", h.Header.PeerPort, err))
	}

	if port < 1 || port > 65535 {
		return (fmt.Errorf("given port out of range: %d", port))
	}
	return nil
}

func newHandshake(conf *Configuration) *Handshake {
	hs := new(Handshake)
	hs.Header = V9Header{
		Network:  conf.Network,
		Version:  conf.ProtocolVersion,
		Type:     TypeHandshake,
		NodeID:   uint64(conf.NodeID),
		PeerPort: conf.ListenPort,
		AppHash:  "NetworkMessage",
		AppType:  "Network",
	}
	hs.SetPayload([]byte("Handshake"))
	return hs
}

func (hs *Handshake) SetPayload(payload []byte) {
	hs.Payload = payload
	hs.Header.Crc32 = crc32.Checksum(hs.Payload, crcTable)
	hs.Header.Length = uint32(len(hs.Payload))
}
