package p2p

import "testing"

func TestPeerMap_Add(t *testing.T) {

	pm := NewPeerMap()

	type args struct {
		p *Peer
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
	}{
		{"simple peer add", pm, args{&Peer{Address: "a", Port: "1", Hash: "a"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arg := tt.args.p
			tt.pm.Add(arg)
			if tt.pm.byHash[arg.Hash] != arg {
				t.Errorf("Failed to save by Hash")
			}

			if list, ok := tt.pm.byIP[arg.Address]; !ok {
				t.Errorf("Failed to save peer to the list by ip")
			} else if p, b := list.HasIPPort(arg.Address, arg.Port); !b || p != arg {
				t.Errorf("Failed to find peer inside list")
			}
		})
	}
}
