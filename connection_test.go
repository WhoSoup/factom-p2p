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

	incoming1 := make(chan *Parcel, StandardChannelSize)
	incoming2 := make(chan *Parcel, StandardChannelSize)
	config := DefaultP2PConfiguration()
	config.ReadDeadline = time.Second
	config.WriteDeadline = time.Second

	pipe1, pipe2 := net.Pipe()

	// test 1 write garbage
	con1 := NewConnection(pipe1, config, incoming1)
	con2 := NewConnection(pipe2, config, incoming2)

	con1.Start()       // start via Start()
	go con2.readLoop() // start "manually"
	go con2.sendLoop()

	var parcels []*Parcel
	for i := 0; i < testParcels; i++ {
		p := NewParcel(0, []byte(fmt.Sprintf("%d", i)))
		parcels = append(parcels, p)
	}

	for _, p := range parcels {
		con1.Outgoing <- p
	}

	fmt.Println("Sent garbage parcels")

	for len(con1.Outgoing) > 0 { // wait until everything is sent
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

	if len(con2.Shutdown) == 0 {
		t.Error("Expected error in connection 2 after writing garbage, got nothing")
	} else {
		e := <-con2.Shutdown
		if e.Error() != "gob: type mismatch in decoder: want struct type p2p.Parcel; got non-struct" {
			t.Errorf("Expected gob type mismatch error on Connection2, got: %s", e.Error())
		}
	}
	if len(con1.Shutdown) == 0 {
		t.Fatal("Expected error in connection 1 after writing garbage, got nothing")
	} else {
		e := <-con1.Shutdown
		if e != io.EOF {
			t.Errorf("Expected EOF on Connection1, got: %s", e.Error())
		}
	}
}
