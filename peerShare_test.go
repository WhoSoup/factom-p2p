package p2p

import (
	"testing"
)

func TestPeerShare_String(t *testing.T) {
	tests := []struct {
		name string
		ps   PeerShare
		want string
	}{
		{"normal", PeerShare{"127.0.0.1", "8088", 0}, "127.0.0.1:8088"},
		{"no addr", PeerShare{"", "8088", 0}, ":8088"},
		{"no port", PeerShare{"127.0.0.1", "", 0}, "127.0.0.1:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.String(); got != tt.want {
				t.Errorf("PeerShare.ConnectAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeerShare_Verify(t *testing.T) {
	tests := []struct {
		name string
		ps   PeerShare
		want bool
	}{
		{"valid address", PeerShare{"127.0.0.1", "80", 0}, true},
		{"no addr", PeerShare{"", "8088", 0}, false},
		{"no port", PeerShare{"127.0.0.1", "", 0}, false},
		{"zero port", PeerShare{"127.0.0.1", "0", 0}, false},
		{"nonnumeric port", PeerShare{"127.0.0.1", "eighty", 0}, false},
		{"nonnumeric port", PeerShare{"127.0.0.1", "80th", 0}, false},
		{"hostname", PeerShare{"factom.fct", "8088", 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.Verify(); got != tt.want {
				t.Errorf("PeerShare.Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}
