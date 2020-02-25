package p2p

import (
	"encoding/gob"
	"net"
	"reflect"
	"testing"
	"time"
)

func Test_ProtocolV9(t *testing.T) {
	conf := DefaultP2PConfiguration()
	conf.Network = NewNetworkID("test")
	conf.ProtocolVersion = 9
	conf.NodeID = 1
	conf.ListenPort = "8999"

	A, B := net.Pipe()
	A.SetDeadline(time.Now().Add(time.Millisecond * 50))
	B.SetDeadline(time.Now().Add(time.Millisecond * 50))

	encoderA := gob.NewEncoder(A)
	decoderA := gob.NewDecoder(A)
	protA := newProtocolV9(conf.Network, conf.NodeID, conf.ListenPort, decoderA, encoderA)

	encoderB := gob.NewEncoder(B)
	decoderB := gob.NewDecoder(B)
	protB := newProtocolV9(conf.Network, conf.NodeID, conf.ListenPort, decoderB, encoderB)

	hs := newHandshake(&conf, 123)

	testParcel := newParcel(TypeMessage, []byte("test"))

	go func() {
		if err := protA.SendHandshake(hs); err != nil {
			t.Error(err)
		}
		if err := protA.Send(testParcel); err != nil {
			t.Error(err)
		}
	}()

	hsReply, err := protB.ReadHandshake()
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(hs, hsReply) {
		t.Errorf("two handshakes not the same. sent = %+v, received = %+v", hs, hsReply)
	}

	reply, err := protB.Receive()
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(testParcel, reply) {
		t.Errorf("two parcels not the same. sent = %+v, received = %+v", testParcel, reply)
	}

}
