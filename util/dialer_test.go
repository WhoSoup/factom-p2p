package util

import (
	"testing"
	"time"
)

func TestDialer(t *testing.T) {
	ip := IP{Address: "127.255.255.254", Port: "65535"}

	d := NewDialer("127.0.0.1", time.Millisecond*50, time.Millisecond*25, 2)

	if !d.CanDial(ip) {
		t.Error("failed to connect first time")
	}
	d.Dial(ip)
	if d.CanDial(ip) {
		t.Error("can dial during blocking interval")
	}

	time.Sleep(time.Millisecond * 50)

	if !d.CanDial(ip) {
		t.Error("failed to connect second time")
	}
	d.Dial(ip)
	if d.CanDial(ip) {
		t.Error("can dial during second blocking interval")
	}

	time.Sleep(time.Millisecond * 50)

	if d.CanDial(ip) {
		t.Error("can dial even though attempts are reached")
	}
}
