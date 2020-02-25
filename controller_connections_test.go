package p2p

import (
	"encoding/gob"
	"net"
	"testing"
	"time"
)

func Test_controller_detectProtocolV9(t *testing.T) {
	conf := DefaultP2PConfiguration()
	conf.Network = NewNetworkID("test")
	conf.ProtocolVersion = 9
	conf.NodeID = 1
	conf.ListenPort = "8999"

	A, B := net.Pipe()
	A.SetDeadline(time.Now().Add(time.Millisecond * 50))
	B.SetDeadline(time.Now().Add(time.Millisecond * 50))

	c := new(controller)
	c.net = new(Network)
	c.net.conf = &conf
	c.net.instanceID = 666

	go func() {
		encoder := gob.NewEncoder(B)
		decoder := gob.NewDecoder(B)
		v9prot := newProtocolV9(conf.Network, conf.NodeID, conf.ListenPort, decoder, encoder)
		hs := newHandshake(&conf, 123)

		if err := v9prot.SendHandshake(hs); err != nil {
			t.Error(err)
		}

		testp := newParcel(TypeMessage, []byte("foo"))
		if err := v9prot.Send(testp); err != nil {
			t.Error(err)
		}
	}()

	prot, hs, err := c.detectProtocol(A)
	if err != nil {
		t.Error(err)
	}

	t.Error(prot.Version())
	t.Error(hs)

	p, err := prot.Receive()
	if err != nil {
		t.Error(err)
	}

	t.Error(string(p.Payload))
}

func Test_controller_detectProtocolV10(t *testing.T) {
	conf := DefaultP2PConfiguration()
	conf.Network = NewNetworkID("test")
	conf.ProtocolVersion = 10
	conf.NodeID = 1
	conf.ListenPort = "8999"

	A, B := net.Pipe()
	A.SetDeadline(time.Now().Add(time.Millisecond * 50))
	B.SetDeadline(time.Now().Add(time.Millisecond * 50))

	c := new(controller)
	c.net = new(Network)
	c.net.conf = &conf
	c.net.instanceID = 666

	go func() {
		encoder := gob.NewEncoder(B)
		decoder := gob.NewDecoder(B)
		v10prot := newProtocolV10(decoder, encoder)
		hs := newHandshake(&conf, 123)

		if err := v10prot.SendHandshake(hs); err != nil {
			t.Error(err)
		}

		testp := newParcel(TypeMessage, []byte("foo"))
		if err := v10prot.Send(testp); err != nil {
			t.Error(err)
		}
	}()

	prot, hs, err := c.detectProtocol(A)
	if err != nil {
		t.Error(err)
	}

	t.Error(prot.Version())
	t.Error(hs)

	p, err := prot.Receive()
	if err != nil {
		t.Error(err)
	}

	t.Error(string(p.Payload))

}
