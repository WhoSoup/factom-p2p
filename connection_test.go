package p2p

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestConnection(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	testParcels := 50

	incoming1 := NewParcelChannel(5000)
	incoming2 := NewParcelChannel(5000)
	config := DefaultP2PConfiguration()
	config.ReadDeadline = time.Second
	config.WriteDeadline = time.Second

	netw := &Network{conf: &config}

	pipe1, pipe2 := net.Pipe()

	// test 1 write garbage
	con1 := NewConnection("incoming1", pipe1, incoming1, netw, true)
	con2 := NewConnection("incoming2", pipe2, incoming2, netw, false)

	con1.Start()       // start via Start()
	go con2.readLoop() // start "manually"
	go con2.sendLoop()

	var parcels []*Parcel
	for i := 0; i < testParcels; i++ {
		p := NewParcel(0, []byte(fmt.Sprintf("%d", i)))
		parcels = append(parcels, p)
	}

	for _, p := range parcels {
		con1.Send.Send(p)
	}

	fmt.Println("Sent garbage parcels")

	for len(con1.Send) > 0 { // wait until everything is sent
		time.Sleep(time.Millisecond * 10)
	}

	var reply []*Parcel
	for len(incoming2) > 0 {
		reply = append(reply, <-incoming2)
	}

	if len(reply) != len(parcels) {
		t.Errorf("some parcels dropped. sent = %d, received = %d", len(parcels), len(reply))
	}

	for i := range reply {
		if bytes.Compare(reply[i].Payload, parcels[i].Payload) != 0 {
			t.Errorf("received parcels out of order. expected %s got %s", parcels[i].Payload, reply[i].Payload)
		}
	}

	// push NON-parcel into the system
	con1.encoder.Encode([]byte("garbage"))

	time.Sleep(time.Millisecond * 50)

	if len(con2.Error) == 0 {
		t.Error("Expected error in connection 2 after writing garbage, got nothing")
	} else {
		e := <-con2.Error
		if e.Error() != "gob: type mismatch in decoder: want struct type p2p.Parcel; got non-struct" {
			t.Errorf("Expected gob type mismatch error on Connection2, got: %s", e.Error())
		}
	}
	if len(con1.Error) == 0 {
		t.Fatal("Expected error in connection 1 after writing garbage, got nothing")
	} else {
		e := <-con1.Error
		if e != io.EOF {
			t.Errorf("Expected EOF on Connection1, got: %s", e.Error())
		}
	}
}
