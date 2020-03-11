package p2p

import (
	"bytes"
	"encoding/gob"
	"net"
	"testing"
	"time"
)

func TestPeer_sendLoop(t *testing.T) {
	A, B := net.Pipe()

	p := new(Peer)
	p.net = testNetworkHarness(t)

	p.logger = packageLogger
	p.conn = A
	p.prot = newProtocolV10(gob.NewDecoder(A), gob.NewEncoder(A))
	start := time.Now()
	p.lastSend = start

	done := make(chan bool, 2)
	p.stop = make(chan bool, 1)

	p.send = newParcelChannel(128)
	go func() {
		p.sendLoop()
		done <- true
	}()

	p.Send(nil)

	parcels := make([]*Parcel, 128)
	for i := range parcels {
		parcels[i] = testRandomParcel()
		added, dropped := p.send.Send(parcels[i])
		if !added || dropped > 0 {
			t.Errorf("failed to add parcel %d. added = %v, dropped = %d", i, added, dropped)
		}
	}

	time.Sleep(time.Millisecond * 50)

	if p.LastSendAge() > time.Since(start) {
		t.Errorf("peer.lastSend did not update")
	}

	go func() {
		dec := gob.NewDecoder(B)
		for i := 0; i < len(parcels); i++ {
			v10msg := new(V10Msg)
			dec.Decode(v10msg)
			if v10msg.Type != parcels[i].ptype || !bytes.Equal(v10msg.Payload, parcels[i].Payload) {
				t.Errorf("parcel %d didn't arrive the same. got = (%s, %x), want = (%s, %x)", i, v10msg.Type, v10msg.Payload, parcels[i].ptype, parcels[i].Payload)
			}
		}
		p.Stop()
	}()

	<-done
	update := <-p.net.controller.peerStatus
	if update.peer != p {
		t.Errorf("peerStatus: received unknown peer %+v", update.peer)
	}
	if update.online {
		t.Errorf("peerStatus: received online signal")
	}
}

func TestPeer_readLoop(t *testing.T) {
	A, B := net.Pipe()

	p := new(Peer)
	p.Hash = "unit test peer"
	p.conn = A
	p.net = testNetworkHarness(t)
	p.logger = packageLogger
	p.stop = make(chan bool, 1)
	p.prot = newProtocolV10(gob.NewDecoder(A), gob.NewEncoder(A))

	start := time.Now()
	p.lastReceive = start

	parcels := make([]*Parcel, p.net.conf.ChannelCapacity*2)
	for i := range parcels {
		parcels[i] = testRandomParcel()
	}

	done := make(chan bool, 1)
	stopped := make(chan bool, 1)

	// check delivery
	go func() {
		for i := 0; i < len(parcels); i++ {
			pp := <-p.net.controller.peerData
			if pp.peer.Hash != p.Hash {
				t.Errorf("peerData: parcel %d received has wrong peer. got = %s, want = %s", i, pp.peer.Hash, p.Hash)
			}
			if pp.parcel.ptype != parcels[i].ptype || !bytes.Equal(pp.parcel.Payload, parcels[i].Payload) {
				t.Errorf("peerData: parcel %d differs. got = (%s, %x), want = (%s, %x)", i, pp.parcel.ptype, pp.parcel.Payload, parcels[i].ptype, parcels[i].Payload)
			}
		}
		done <- true
	}()

	// run readloop
	go func() {
		p.readLoop()
		stopped <- true
	}()

	// send messages
	go func() {
		enc := gob.NewEncoder(B)
		for _, p := range parcels {
			msg := new(V10Msg)
			msg.Type = p.ptype
			msg.Payload = p.Payload
			B.SetWriteDeadline(time.Now().Add(time.Millisecond * 50))
			if err := enc.Encode(msg); err != nil {
				t.Errorf("error encoding: %v", err)
			}
		}
	}()

	<-done // all parcels were sent to the controller
	p.Stop()
	<-stopped // the readloop exited successfully

	if p.lastReceive.Equal(start) {
		t.Errorf("peer.lastReceive did not update. lastReceive = %s, start = %s", p.lastReceive, start)
	}

	update := <-p.net.controller.peerStatus
	if update.peer != p {
		t.Errorf("peerStatus: received unknown peer %+v", update.peer)
	}
	if update.online {
		t.Errorf("peerStatus: received online signal")
	}
}
