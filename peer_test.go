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
	p.conn = A
	p.net = new(Network)
	c := DefaultP2PConfiguration()
	p.net.conf = &c
	p.prot = newProtocolV10(gob.NewDecoder(A), gob.NewEncoder(A))
	start := time.Now()
	p.lastSend = start

	done := make(chan bool, 2)
	p.stop = make(chan bool, 1)
	p.net.stopper = make(chan interface{}, 1)

	p.send = newParcelChannel(128)
	go func() {
		p.sendLoop()
		done <- true
	}()

	p.logger = packageLogger

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
		close(p.stop)
	}()

	<-done
}
