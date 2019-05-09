package p2p

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net"

	"github.com/gogo/protobuf/proto"
	"github.com/whosoup/factom-p2p/p11"
)

var _ Protocol = (*ProtocolV11)(nil)

type ProtocolV11 struct {
	net  *Network
	conn net.Conn
	peer *Peer
	r    *bufio.Reader
	buf  *bytes.Buffer
}

func (v11 *ProtocolV11) Init(peer *Peer, conn net.Conn, r *bufio.Reader) {
	v11.peer = peer
	v11.net = peer.net
	v11.conn = conn
	v11.r = r
	v11.buf = new(bytes.Buffer)
}

func (v11 *ProtocolV11) Send(p *Parcel) error {
	var msg p11.V11Msg
	msg.Type = uint32(p.Type)
	msg.Crc32 = crc32.Checksum(p.Payload, crcTable)
	msg.Payload = p.Payload
	b, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	l := uint32(len(b))
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], l)

	written, err := v11.conn.Write(buf[:])
	if err != nil {
		return err
	}
	if written != 4 {
		return fmt.Errorf("unable to write size info to connection, %d of %d written", written, 4)
	}

	written, err = v11.conn.Write(b)
	if err != nil {
		return err
	}

	if written != len(b) {
		return fmt.Errorf("unable to write buffer to connection, %d of %d written", written, len(b))
	}

	return nil
}

func (v11 *ProtocolV11) Version() string {
	return "11"
}

func (v11 *ProtocolV11) ReadFull(n int) ([]byte, error) {
	b := make([]byte, n)
	z, err := io.ReadFull(v11.r, b)
	if z < n {
		return nil, err
	}
	/*	read := 0
		for n > 0 {
			read, err := v11.conn.Read(b[read:])
			if err != nil {
				return b, err
			}

			if read == 0 {
				return b, fmt.Errorf("reading zero bytes")
			}

			n -= read
		}*/
	return b, nil
}

func (v11 *ProtocolV11) Receive() (*Parcel, error) {
	sizebuf, err := v11.ReadFull(4)
	if err != nil {
		return nil, err
	}
	size := binary.LittleEndian.Uint32(sizebuf[:])

	data, err := v11.ReadFull(int(size))
	if err != nil {
		return nil, err
	}

	var msg p11.V11Msg
	err = proto.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	if len(msg.Payload) == 0 {
		return nil, fmt.Errorf("nul payload")
	}

	csum := crc32.Checksum(msg.Payload, crcTable)
	if csum != msg.Crc32 {
		return nil, fmt.Errorf("invalid checksum")
	}

	p := newParcel(ParcelType(msg.Type), msg.Payload)
	p.Address = v11.conn.RemoteAddr().String()
	return p, nil
}
