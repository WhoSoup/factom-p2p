package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
)

// V11MaxParcelSize limits the amount of ram allocated for a parcel to 128 MiB
const V11MaxParcelSize = 134217728

// V11Signature is the 4-byte sequence that indicates the remote connection wants to use V11
var V11Signature = []byte{0xfa, 0xfa, 0xfa, 0xfa}

var _ Protocol = (*ProtocolV11)(nil)

type ProtocolV11 struct {
	rw io.ReadWriter
}

func newProtocolV11(rw io.ReadWriter) *ProtocolV11 {
	v11 := new(ProtocolV11)
	v11.rw = rw
	return v11
}

func (v11 *ProtocolV11) SendHandshake(*Handshake) error     { return nil }
func (v11 *ProtocolV11) ReadHandshake() (*Handshake, error) { return nil, nil }

func (v11 *ProtocolV11) writeCheck(data []byte) error {
	if n, err := v11.rw.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("unable to write data (%d of %d)", n, len(data))
	}
	return nil
}

func (v11 *ProtocolV11) readCheck(data []byte) error {
	if n, err := v11.rw.Read(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("unable to read buffer (%d of %d)", n, len(data))
	}
	return nil
}

func (v11 *ProtocolV11) Send(p *Parcel) error {
	buf := make([]byte, 4)

	msg := new(V11Msg)
	msg.Type = uint32(p.Type)
	msg.Payload = p.Payload

	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	binary.BigEndian.PutUint32(buf, uint32(len(data)))

	if err := v11.writeCheck(buf); err != nil {
		return err
	}

	return v11.writeCheck(data)
}

func (v11 *ProtocolV11) Receive() (*Parcel, error) {
	buf := make([]byte, 4)
	if err := v11.readCheck(buf); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(buf)

	if size > V11MaxParcelSize {
		return nil, fmt.Errorf("peer attempted to send a parcel of size %d (max %d)", size, V11MaxParcelSize)
	}

	data := make([]byte, size)
	if err := v11.readCheck(data); err != nil {
		return nil, err
	}

	msg := new(V11Msg)
	if err := msg.Unmarshal(data); err != nil {
		return nil, err
	}

	// type validity is checked in parcel.Valid
	return newParcel(ParcelType(msg.Type), msg.Payload), nil
}

func (v11 *ProtocolV11) Version() string {
	return "11"
}

func (v11 *ProtocolV11) MakePeerShare(ps []Endpoint) ([]byte, error) {
	return nil, nil
}

func (v11 *ProtocolV11) ParsePeerShare(payload []byte) ([]Endpoint, error) {
	return nil, nil
}
