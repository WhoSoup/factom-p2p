package p2p

import "testing"

func TestHandshake_Valid(t *testing.T) {
	conf := DefaultP2PConfiguration()

	var handshakes []*Handshake
	for i := 0; i < 13; i++ {
		hs := newHandshake(&conf, 0)
		hs.NodeID++
		handshakes = append(handshakes, hs)
	}
	// 0 is default payload
	handshakes[1].Version = 2 // incompatible version
	handshakes[2].Network = 0xf00
	handshakes[3].ListenPort = "foo"
	handshakes[4].ListenPort = ""
	handshakes[5].ListenPort = "0"
	handshakes[6].ListenPort = "900000"

	type args struct {
		conf *Configuration
	}
	tests := []struct {
		name    string
		h       *Handshake
		args    args
		wantErr bool
	}{
		{"default (valid)", handshakes[0], args{&conf}, false},
		{"wrong version", handshakes[1], args{&conf}, true},
		{"wrong network", handshakes[2], args{&conf}, true},
		{"unparseable port", handshakes[3], args{&conf}, true},
		{"empty port", handshakes[4], args{&conf}, true},
		{"zero port", handshakes[5], args{&conf}, true},
		{"too high port", handshakes[6], args{&conf}, true},
		{"nil payload", handshakes[7], args{&conf}, true},
		{"no payload initialized", handshakes[8], args{&conf}, true},
		{"wrong payload length", handshakes[9], args{&conf}, true},
		{"wrong payload crc", handshakes[10], args{&conf}, true},
		{"wrong payload bytes", handshakes[11], args{&conf}, true},
		{"no peer address", handshakes[12], args{&conf}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.Valid(tt.args.conf, 0); (err != nil) != tt.wantErr {
				t.Errorf("Handshake.Valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
