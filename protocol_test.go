package p2p

import (
	"encoding/gob"
	"io"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"
)

func testRandomParcel() *Parcel {
	p := new(Parcel)
	p.Type = ParcelType(rand.Intn(len(typeStrings)))
	p.Address = ""                              // address is internal in prot10+
	p.Payload = make([]byte, 1+rand.Intn(8191)) // 1 byte - 8KiB
	rand.Read(p.Payload)
	return p
}

func testParcelSendReceive(t *testing.T, protf func(io.ReadWriter) Protocol) {
	parcels := make([]*Parcel, 128)
	for i := range parcels {
		parcels[i] = testRandomParcel()
	}

	A, B := net.Pipe()
	defer A.Close()
	defer B.Close()
	sender := protf(A)
	receiver := protf(B)

	dl := time.Now().Add(time.Millisecond * 500)
	A.SetDeadline(dl)
	B.SetDeadline(dl)

	go func() {
		for _, p := range parcels {
			if err := sender.Send(p); err != nil {
				t.Errorf("prot %s: error sending %+v: %v", sender, p, err)
			}
		}
	}()

	for i := range parcels {
		p, err := receiver.Receive()
		if err != nil {
			t.Errorf("prot %s: error receiving %+v: %v", receiver, parcels[i], p)
			continue
		}

		if !reflect.DeepEqual(parcels[i], p) { // test parcels are made without an address
			t.Errorf("prot %s: received wrong message. want = %+v, got = %+v", receiver, parcels[i], p)
		}
	}
}

func testProtV9(rw io.ReadWriter) Protocol {
	decoder := gob.NewDecoder(rw)
	encoder := gob.NewEncoder(rw)
	return newProtocolV9(TestNet, 0x666, "9999", decoder, encoder)
}

func testProtV10(rw io.ReadWriter) Protocol {
	decoder := gob.NewDecoder(rw)
	encoder := gob.NewEncoder(rw)
	return newProtocolV10(decoder, encoder)
}

func testProtV11(rw io.ReadWriter) Protocol {
	return newProtocolV11(rw)
}

func TestParcelSendReceive(t *testing.T) {
	testParcelSendReceive(t, testProtV9)
	testParcelSendReceive(t, testProtV10)
	testParcelSendReceive(t, testProtV11)
}

func testHandshake(t *testing.T, protf func(io.ReadWriter) Protocol) {
	A, B := net.Pipe()
	defer A.Close()
	defer B.Close()

	sender := protf(A)
	receiver := protf(B)

	hs := testRandomHandshake()

	go func() {
		if err := sender.SendHandshake(hs); err != nil {
			t.Errorf("prot %s: send handshake err: %v", sender, err)
		}
	}()

	if reply, err := receiver.ReadHandshake(); err != nil {
		t.Errorf("prot %s: read handshake err: %v", receiver, err)
	} else {
		if len(reply.Alternatives) == 0 {
			reply.Alternatives = nil
		}
		if len(hs.Alternatives) == 0 {
			hs.Alternatives = nil
		}
		if !reflect.DeepEqual(hs, reply) {
			t.Errorf("prot %s: handshake different. sent = %+v, got = %+v. %+v", receiver, hs, reply, reflect.DeepEqual(hs.Alternatives, reply.Alternatives))
		}
	}
}

func TestProtocol_Handshake(t *testing.T) {
	for i := 0; i < 128; i++ {
		testHandshake(t, testProtV9)
		testHandshake(t, testProtV11)
	}
}
