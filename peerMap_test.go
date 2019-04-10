package p2p

import (
	"fmt"
	"reflect"
	"testing"
)

func basicPM() (*PeerMap, []*Peer) {
	pm := NewPeerMap()
	list := testPeers()
	for _, p := range list {
		pm.Add(p)
	}
	return pm, list
}

func testPeers() []*Peer {
	return []*Peer{
		&Peer{Address: "a", Port: "1", Hash: "a1"},
		&Peer{Address: "b", Port: "2", Hash: "b2"},
		&Peer{Address: "c", Port: "3", Hash: "c3"},
		&Peer{Address: "d", Port: "4", Hash: "d4"},
	}
}

func TestPeerMap_Add(t *testing.T) {
	pm := NewPeerMap()
	pm.Add(nil)

	list := testPeers()

	type args struct {
		p *Peer
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
	}{
		{"simple peer add 1", pm, args{list[0]}},
		{"simple peer add 2", pm, args{list[1]}},
		{"simple peer add 3", pm, args{list[2]}},
		{"simple peer add 4", pm, args{list[3]}},
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
			} else if p, b := list.Search(arg.Address, arg.Port); !b || p != arg {
				t.Errorf("Failed to find peer inside list")
			}
		})
	}
}

func TestPeerMap_Remove(t *testing.T) {
	pm, list := basicPM()

	type args struct {
		p *Peer
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
	}{
		{"normal1", pm, args{list[0]}},
		{"normal2", pm, args{list[1]}},
		{"normal3", pm, args{list[2]}},
		{"normal4", pm, args{list[3]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// check if peer is in list
			p := tt.args.p
			if ok := tt.pm.Has(p.Hash); tt.args.p != nil && !ok {
				t.Errorf("Trying to remove nonexistent peer")
			}
			tt.pm.Remove(p)

			if _, ok := tt.pm.byHash[p.Hash]; ok {
				t.Errorf("byHash still has hash %s after removal", p.Hash)
			}

			if list, ok := tt.pm.byIP[p.Address]; ok {
				if _, e := list.Search(p.Address, p.Port); e {
					t.Errorf("byIP still contains peer %s after removal", p.Hash)
				}
			}

			if _, e := tt.pm.bySlice.Search(p.Address, p.Port); e {
				t.Errorf("bySlice still contains peer %s after removal", p.Hash)
			}
		})
	}

	if len(pm.byHash) > 0 {
		t.Errorf("map still contains entries. size = %d", len(pm.byHash))
		for loc, p := range pm.byHash {
			fmt.Println(loc, "=", p.Hash)
		}
	}

	pm.Remove(nil)
	pm.Remove(&Peer{Address: "a", Port: "1", Hash: "a"})
}

func TestPeerMap_Has(t *testing.T) {
	pm, _ := basicPM()
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
		want bool
	}{
		{"normal", pm, args{"a1"}, true},
		{"nonexistent", pm, args{"a2"}, false},
		{"empty", pm, args{""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pm.Has(tt.args.hash); got != tt.want {
				t.Errorf("PeerMap.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeerMap_Get(t *testing.T) {
	pm, list := basicPM()

	type args struct {
		hash string
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
		want *Peer
	}{
		{"nil", pm, args{""}, nil},
		{"nonexistent", pm, args{"doesn't exist"}, nil},
	}

	for _, p := range list {
		tests = append(tests, struct {
			name string
			pm   *PeerMap
			args args
			want *Peer
		}{fmt.Sprintf("verify %s", p.Hash), pm, args{p.Hash}, p})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pm.Get(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PeerMap.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeerMap_Slice(t *testing.T) {
	pm, list := basicPM()
	slice := pm.Slice()

	pm.Remove(list[0])

	// code mostly same as in peerList_test
	found := 0
Outer:
	for _, control := range list {
		for _, a := range slice {
			if a == nil {
				t.Fatalf("nil-pointer in slice")
			}
			if a == control {
				found++
				continue Outer
			}
		}
	}

	if found != len(list) {
		t.Errorf("List did not have all of the original set. Found %d of %d.", found, len(list))
	}
}

func TestPeerMap_Search(t *testing.T) {
	pm, list := basicPM()

	type args struct {
		addr string
		port string
	}
	tests := []struct {
		name  string
		pm    *PeerMap
		args  args
		want  *Peer
		want1 bool
	}{
		{"normal", pm, args{"a", "1"}, list[0], true},
		{"wrong port", pm, args{"a", "2"}, nil, false},
		{"normal 2", pm, args{"b", "2"}, list[1], true},
		{"blank both", pm, args{"", ""}, nil, false},
		{"blank addr", pm, args{"", "1"}, nil, false},
		{"blank port", pm, args{"a", ""}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.pm.Search(tt.args.addr, tt.args.port)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PeerMap.Search() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("PeerMap.Search() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestPeerMap_IsConnected(t *testing.T) {
	pm, _ := basicPM()
	pm.Add(&Peer{Address: "a", Port: "2", Hash: "a2", state: Online})
	pm.Add(&Peer{Address: "b", Port: "1", Hash: "b1", state: Connecting})

	type args struct {
		address string
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
		want bool
	}{
		{"working online", pm, args{"a"}, true},
		{"working connecting", pm, args{"b"}, true},
		{"working offline", pm, args{"c"}, false},
		{"working empty", pm, args{""}, false},
		{"working nonexistent", pm, args{"abc"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pm.IsConnected(tt.args.address); got != tt.want {
				t.Errorf("PeerMap.IsConnected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPeerMap_ConnectedCount(t *testing.T) {
	pm, _ := basicPM()
	pm.Add(&Peer{Address: "a", Port: "10", Hash: "a10", state: Online})
	pm.Add(&Peer{Address: "b", Port: "10", Hash: "b10", state: Connecting})
	pm.Add(&Peer{Address: "c", Port: "10", Hash: "c10", state: Online})
	pm.Add(&Peer{Address: "c", Port: "11", Hash: "c11", state: Connecting})
	pm.Add(&Peer{Address: "c", Port: "12", Hash: "c12", state: Online})
	type args struct {
		address string
	}
	tests := []struct {
		name string
		pm   *PeerMap
		args args
		want int
	}{
		{"working none", pm, args{"abc"}, 0},
		{"working one", pm, args{"a"}, 1},
		{"working connecting", pm, args{"b"}, 1},
		{"working mixed", pm, args{"c"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pm.ConnectedCount(tt.args.address); got != tt.want {
				t.Errorf("PeerMap.ConnectedCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
