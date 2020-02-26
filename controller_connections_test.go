package p2p

import (
	"bytes"
	"net"
	"reflect"
	"testing"
	"time"
)

func _detectProtocol(t *testing.T, version uint16) {
	conf := DefaultP2PConfiguration()
	conf.Network = NewNetworkID("test")
	conf.ProtocolVersion = version
	conf.NodeID = 1
	conf.ListenPort = "8999"

	a, b := net.Pipe()
	A := NewMetricsReadWriter(a)
	B := NewMetricsReadWriter(b)
	a.SetDeadline(time.Now().Add(time.Millisecond * 50))
	b.SetDeadline(time.Now().Add(time.Millisecond * 50))

	c := new(controller)
	c.net = new(Network)
	c.net.conf = &conf
	c.net.instanceID = 666

	sendprot := c.selectProtocol(B)
	sendshake := newHandshake(&conf, 123)

	go func() {
		if err := sendprot.SendHandshake(sendshake); err != nil {
			t.Error(err)
		}

		testp := newParcel(TypeMessage, []byte("foo"))
		if err := sendprot.Send(testp); err != nil {
			t.Error(err)
		}
	}()

	prot, hs, err := c.detectProtocolFromFirstMessage(A)
	if err != nil {
		t.Error(err)
	}

	if prot.Version() != version {
		t.Errorf("version mismatch. want = %d, got = %s", version, prot)
	}

	if !reflect.DeepEqual(sendshake, hs) {
		t.Errorf("handshake differs. want = %+v, got = %+v", sendshake, hs)
	}

	p, err := prot.Receive()
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(p.Payload, []byte("foo")) {
		t.Errorf("parcel didn't decode properly. want = %x, got = %x", []byte("foo"), p.Payload)
	}
}

func Test_controller_detectProtocol(t *testing.T) {
	_detectProtocol(t, 9)
	_detectProtocol(t, 10)
	_detectProtocol(t, 11)
}
