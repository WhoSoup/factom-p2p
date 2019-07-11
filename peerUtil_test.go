package p2p

import (
	"testing"
)

/*
func createSet(locations ...uint32) *PeerDistance {
	pd := new(PeerDistance)
	pd.Pivot = locations[0]
	for _, l := range locations[1:] {
		pd.Sorted = append(pd.Sorted, IP{Location: l})
	}
	return pd
}

func checkSet(pd *PeerDistance) bool {
	prev := uint32(0)
	for _, l := range pd.Sorted {
		if uintDistance(pd.Pivot, l.Location) < prev {
			fmt.Printf("Failed location %d: %d < %d\n", l.Location, prev, uintDistance(pd.Pivot, l.Location))
			return false
		}
	}
	return true
}

func simpleTest(t *testing.T, name string, set ...uint32) {
	s := createSet(set...)
	sort.Sort(s)
	if !checkSet(s) {
		t.Errorf("%s not sorted right", name)
	}
	fmt.Printf("Test %s, Pivot: %d, list =", name, s.Pivot)
	for _, p := range s.Sorted {
		fmt.Printf(" %d", p.Location)
	}
	fmt.Println()
}

func Test_sorting(t *testing.T) {
	simpleTest(t, "simple blank", 0)
	simpleTest(t, "simple 0-5", 0, 1, 2, 3, 4, 5)
	simpleTest(t, "simple alt", 10, 11, 9, 12, 8, 13)
	simpleTest(t, "reverse", 0, 5, 4, 3, 2, 1)
	simpleTest(t, "duplicates", 0, 1, 1, 2, 2, 2, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4)
	simpleTest(t, "big", 4294967295/2, 4294967295/500, 4294967295/2, 4294967295, 5000, 800000)

	r := rand.New(rand.NewSource(0))
	set := make([]uint32, 1)
	set[0] = 4294967295 / 2
	for i := 0; i < 5000; i++ {
		set = append(set, r.Uint32())
	}
	simpleTest(t, "random", set...)
}

func Test_uintDistance(t *testing.T) {
	type args struct {
		i uint32
		j uint32
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{"0-0", args{0, 0}, 0},
		{"normal", args{10, 5}, 5},
		{"normal reverse", args{5, 10}, 5},
		{"equal", args{25, 25}, 0},
		{"max", args{4294967295, 0}, 4294967295},
		{"max 2", args{4294967295, 4294967294}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uintDistance(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("uintDistance() = %v, want %v", got, tt.want)
			}
		})
	}
}
*/
func TestPeerShare_String(t *testing.T) {
	tests := []struct {
		name string
		ps   PeerShare
		want string
	}{
		{"normal", PeerShare{Address: "127.0.0.1", Port: "8088"}, "127.0.0.1:8088"},
		{"no addr", PeerShare{Address: "", Port: "8088"}, ":8088"},
		{"no port", PeerShare{Address: "127.0.0.1", Port: ""}, "127.0.0.1:"},
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
		{"valid address", PeerShare{Address: "127.0.0.1", Port: "80"}, true},
		{"no addr", PeerShare{Address: "", Port: "8088"}, false},
		{"no port", PeerShare{Address: "127.0.0.1", Port: ""}, false},
		{"zero port", PeerShare{Address: "127.0.0.1", Port: "0"}, false},
		{"nonnumeric port", PeerShare{Address: "127.0.0.1", Port: "eighty"}, false},
		{"nonnumeric port", PeerShare{Address: "127.0.0.1", Port: "80th"}, false},
		{"hostname", PeerShare{Address: "factom.fct", Port: "8088"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ps.Verify(); got != tt.want {
				t.Errorf("PeerShare.Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}
