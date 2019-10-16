package p2p

import (
	"testing"
	"time"
)

func TestDialer(t *testing.T) {
	ep := Endpoint{IP: "127.255.255.254", Port: "65535"}

	d := NewDialer("127.0.0.1", time.Millisecond*50, time.Millisecond*25, time.Minute, 2)

	if !d.CanDial(ep) {
		t.Error("failed to connect first time")
	}
	d.Dial(ep)
	if d.CanDial(ep) {
		t.Error("can dial during blocking interval")
	}

	time.Sleep(time.Millisecond * 50)

	if !d.CanDial(ep) {
		t.Error("failed to connect second time")
	}
	d.Dial(ep)
	if d.CanDial(ep) {
		t.Error("can dial during second blocking interval")
	}

	time.Sleep(time.Millisecond * 50)

	if d.CanDial(ep) {
		t.Error("can dial even though attempts are reached")
	}
}
