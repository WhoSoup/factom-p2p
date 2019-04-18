package p2p

import (
	"reflect"
	"testing"
)

func TestPeerList_Add(t *testing.T) {
	peers := []*Peer{
		&Peer{Address: "a", Port: "1", Hash: "a1"},
		&Peer{Address: "b", Port: "2", Hash: "b2"},
		&Peer{Address: "c", Port: "3", Hash: "c3"},
	}

	list := NewPeerList()
	list.Add(nil) // no effect
	for _, p := range peers {
		list.Add(p)
	}

	if len(list.list) != len(peers) {
		t.Errorf("Not all items were added. Given: %d, Has: %d", len(peers), len(list.list))
	}

	found := 0
Outer:
	for _, a := range list.list {
		for _, control := range peers {
			if a == control {
				found++
				continue Outer
			}
		}
	}

	if found != len(peers) {
		t.Errorf("List has elements that are not present in the original set. Found %d of %d.", found, len(peers))
	}

}

func TestPeerList_Remove(t *testing.T) {
	peers := []*Peer{
		&Peer{Address: "a", Port: "1", Hash: "a1"},
		&Peer{Address: "b", Port: "2", Hash: "b2"},
		&Peer{Address: "c", Port: "3", Hash: "c3"},
		&Peer{Address: "c", Port: "3", Hash: "c3"},
	}
	list := NewPeerList()
	for _, p := range peers {
		list.Add(p)
	}
	for _, p := range peers {
		list.Remove(p)
	}

	if len(list.list) != 0 {
		t.Errorf("list was not cleared of all elements, %d left", len(list.list))
	}

	for _, p := range peers { // should have no effect
		list.Remove(p)
	}

	list.Remove(nil) // has no effect
}

func TestPeerList_Slice(t *testing.T) {
	peers := []*Peer{
		&Peer{Address: "a", Port: "1", Hash: "a1"},
		&Peer{Address: "b", Port: "2", Hash: "b2"},
		&Peer{Address: "c", Port: "3", Hash: "c3"},
	}
	list := NewPeerList()
	for _, p := range peers {
		list.Add(p)
	}

	slice := list.Slice()
	list.Remove(peers[1]) // remove a peer from the original but keep the slice intact

	found := 0
Outer:
	for _, control := range peers {
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

	if found != len(peers) {
		t.Errorf("List did not have all of the original set. Found %d of %d.", found, len(peers))
	}
}

func TestPeerList_Search(t *testing.T) {
	peers := []*Peer{
		&Peer{Address: "a", Port: "1", Hash: "a1"},
		&Peer{Address: "b", Port: "2", Hash: "b2"},
		&Peer{Address: "c", Port: "3", Hash: "c3"},
	}
	list := NewPeerList()
	for _, p := range peers {
		list.Add(p)
	}

	type args struct {
		addr string
		port string
	}
	tests := []struct {
		name  string
		pl    *PeerList
		args  args
		want  *Peer
		want1 bool
	}{
		{"normal", list, args{"a", "1"}, peers[0], true},
		{"wrong port", list, args{"a", "2"}, nil, false},
		{"normal2", list, args{"c", "3"}, peers[2], true},
		{"no port", list, args{"a", ""}, nil, false},
		{"no addr", list, args{"", "1"}, nil, false},
		{"blank", list, args{"", ""}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.pl.Search(tt.args.addr, tt.args.port)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PeerList.Search() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("PeerList.Search() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
