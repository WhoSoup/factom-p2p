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
		{"normal", PeerShare{Address: "127.0.0.1", Port: "8088", QualityScore: 0}, "127.0.0.1:8088"},
		{"no addr", PeerShare{Address: "", Port: "8088", QualityScore: 0}, ":8088"},
		{"no port", PeerShare{Address: "127.0.0.1", Port: "", QualityScore: 0}, "127.0.0.1:"},
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
		{"valid address", PeerShare{Address: "127.0.0.1", Port: "80", QualityScore: 0}, true},
		{"no addr", PeerShare{Address: "", Port: "8088", QualityScore: 0}, false},
		{"no port", PeerShare{Address: "127.0.0.1", Port: "", QualityScore: 0}, false},
		{"zero port", PeerShare{Address: "127.0.0.1", Port: "0", QualityScore: 0}, false},
		{"nonnumeric port", PeerShare{Address: "127.0.0.1", Port: "eighty", QualityScore: 0}, false},
		{"nonnumeric port", PeerShare{Address: "127.0.0.1", Port: "80th", QualityScore: 0}, false},
		{"hostname", PeerShare{Address: "factom.fct", Port: "8088", QualityScore: 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.Verify(); got != tt.want {
				t.Errorf("PeerShare.Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}
