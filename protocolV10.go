package p2p

import (
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"hash/crc32"
)

var _ Protocol = (*ProtocolV10)(nil)

// ProtocolV10 is the protocol introduced by p2p 2.0.
// It is a slimmed down version of V9, reducing overhead
type ProtocolV10 struct {
	decoder *gob.Decoder
	encoder *gob.Encoder
}

// V10Msg is the barebone message
type V10Msg struct {
	Type    ParcelType
	Crc32   uint32
	Payload []byte
}

func newProtocolV10(decoder *gob.Decoder, encoder *gob.Encoder) *ProtocolV10 {
	v10 := new(ProtocolV10)
	v10.decoder = decoder
	v10.encoder = encoder
	return v10
}

func (v10 *ProtocolV10) SendHandshake(h *Handshake) error {
	if h.Type == TypeHandshake {
		h.Type = TypePeerRequest
	}

	var payload []byte
	if len(h.Alternatives) > 0 {
		if data, err := json.Marshal(h.Alternatives); err != nil {
			return err
		} else {
			payload = data
		}
	} else {
		payload = make([]byte, 8)
		binary.LittleEndian.PutUint64(payload, h.Loopback)
	}

	var msg V9Handshake
	msg.Header.Network = h.Network
	msg.Header.Version = 10 // hardcoded
	msg.Header.Type = h.Type
	msg.Header.TargetPeer = ""

	msg.Header.NodeID = uint64(h.NodeID)
	msg.Header.PeerAddress = ""
	msg.Header.PeerPort = h.ListenPort
	msg.Header.AppHash = "NetworkMessage"
	msg.Header.AppType = "Network"

	msg.Payload = payload
	msg.Header.Crc32 = crc32.Checksum(msg.Payload, crcTable)
	msg.Header.Length = uint32(len(msg.Payload))

	return v10.encoder.Encode(&msg)
}

func (v10 *ProtocolV10) ReadHandshake() (*Handshake, error) {
	return nil, fmt.Errorf("V10 doesn't have its own handshake")
	/*msg := new(V9Msg)
	err := v10.decoder.Decode(msg)
	if err != nil {
		return nil, err
	}

	if err = msg.Valid(); err != nil {
		return nil, err
	}

	hs := new(Handshake)
	hs.Type = msg.Header.Type

	if msg.Header.Type == TypeRejectAlternative {
		var alternatives []Endpoint
		if err = json.Unmarshal(msg.Payload, &alternatives); err != nil {
			return nil, err
		}
		hs.Alternatives = alternatives
	} else if len(msg.Payload) == 8 {
		hs.Loopback = binary.BigEndian.Uint64(msg.Payload)
	}
	hs.ListenPort = msg.Header.PeerPort
	hs.Network = msg.Header.Network
	hs.NodeID = uint32(msg.Header.NodeID)
	hs.Version = msg.Header.Version
	return hs, nil*/
}

// Send encodes a Parcel as V10Msg, calculates the crc and encodes it as gob
func (v10 *ProtocolV10) Send(p *Parcel) error {
	var msg V10Msg
	msg.Type = p.Type
	msg.Crc32 = crc32.Checksum(p.Payload, crcTable)
	msg.Payload = p.Payload
	return v10.encoder.Encode(msg)
}

// Version 10
func (v10 *ProtocolV10) Version() string {
	return "10"
}

// Receive converts a V10Msg back to a Parcel
func (v10 *ProtocolV10) Receive() (*Parcel, error) {
	var msg V10Msg
	err := v10.decoder.Decode(&msg)
	if err != nil {
		return nil, err
	}

	if len(msg.Payload) == 0 {
		return nil, fmt.Errorf("null payload")
	}

	csum := crc32.Checksum(msg.Payload, crcTable)
	if csum != msg.Crc32 {
		return nil, fmt.Errorf("invalid checksum")
	}

	p := newParcel(msg.Type, msg.Payload)
	return p, nil
}

// V10Share is an alias of PeerShare
type V10Share Endpoint

// MakePeerShare serializes a list of ips via json
func (v10 *ProtocolV10) MakePeerShare(share []Endpoint) ([]byte, error) {
	var peershare []V10Share
	for _, ep := range share {
		peershare = append(peershare, V10Share{IP: ep.IP, Port: ep.Port})
	}
	return json.Marshal(peershare)
}

// ParsePeerShare parses a peer share payload
func (v10 *ProtocolV10) ParsePeerShare(payload []byte) ([]Endpoint, error) {
	var share []Endpoint
	err := json.Unmarshal(payload, &share)
	return share, err
}
