package p2p

import (
	"testing"
	"time"
)

func testIPs() []IP {
	return []IP{
		IP{Address: "127.0.0.1", Port: "80"},
		IP{Address: "127.0.0.1", Port: "81"},
		IP{Address: "127.0.0.2", Port: "80"},
		IP{Address: "127.0.0.2", Port: "81"},
		IP{Address: "127.0.0.3", Port: "80"}, // twice
		IP{Address: "127.0.0.3", Port: "80"}, // ^
	}
}

func TestEndpoints_Register(t *testing.T) {
	eps := NewEndpoints()
	testips := testIPs()

	before := time.Now()

	for _, ip := range testips {
		eps.Register(ip, "test")
		eps.Register(ip, "test2")
	}

	for _, ip := range testips {
		ep := eps.Ends[ip.String()]
		if ep.IP != ip {
			t.Errorf("ip mismatch, expected %s but got %s", ip, ep.IP)
		}

		seen := ep.Seen
		if seen.Before(before) || time.Now().Before(seen) {
			t.Errorf("seen time mismatch for %s", ip)
		}

		if ep.Source["test"].IsZero() {
			t.Errorf("source 'test' not set for %s", ip)
		}
		if ep.Source["test2"].IsZero() {
			t.Errorf("source 'test2' not set for %s", ip)
		}
	}
}

/*func TestEndpoints_Refresh(t *testing.T) {
	ip := testIPs()[0]
	eps := NewEndpoints()
	eps.Register(ip, "")
	old := eps.Ends[ip.String()].Seen
	time.Sleep(time.Millisecond)
	eps.Refresh(ip)
	if eps.LastSeen(ip).Equal(old) {
		t.Errorf("time did not refresh, stayed the same")
	}
	if eps.LastSeen(ip).Before(old) {
		t.Errorf("went back in time after refresh. zero time = %v", eps.Ends[ip.String()].Seen.IsZero())
	}
}*/

func TestEndpoints_Deregister(t *testing.T) {
	eps := NewEndpoints()
	testips := testIPs()

	for _, ip := range testips {
		eps.Register(ip, "test")
	}

	for _, ip := range testips {
		eps.Deregister(ip)
	}

	if len(eps.IPs()) > 0 {
		t.Errorf("not all endpoints deregistered: %v", eps.IPs())
	}
}

func TestEndpoints_BanAddress(t *testing.T) {
	eps := NewEndpoints()
	ips := testIPs()

	for _, ip := range ips {
		eps.Register(ip, "test")
	}

	now := time.Now()
	eps.BanAddress(ips[0].Address, now)
	eps.BanAddress(ips[2].Address, now)

	if len(eps.IPs()) != 1 {
		t.Errorf("ips didn't get removed after banning 2/3: %v", eps.IPs())
	}

	eps.BanAddress(ips[4].Address, now)
	if len(eps.IPs()) != 0 {
		t.Errorf("ips didn't get removed after banning all: %v", eps.IPs())
	}
}

func TestEndpoints_Banned(t *testing.T) {
	eps := NewEndpoints()
	now := time.Now()
	eps.BanAddress("a", now.Add(-time.Second))
	eps.BanAddress("b", now.Add(time.Second))
	type args struct {
		addr string
	}
	tests := []struct {
		name string
		epm  *Endpoints
		args args
		want bool
	}{
		{"case a", eps, args{"a"}, false},
		{"case b", eps, args{"b"}, true},
		{"case c", eps, args{"c"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.epm.BannedAddress(tt.args.addr); got != tt.want {
				t.Errorf("Endpoints.Banned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpoints_Special(t *testing.T) {
	eps := NewEndpoints()
	ips := testIPs()
	eps.Register(ips[0], "Special")

	type args struct {
		ip IP
	}
	tests := []struct {
		name     string
		epm      *Endpoints
		args     args
		wantIP   bool
		wantAddr bool
	}{ // sp = special
		{"special both", eps, args{ips[0]}, true, true},
		{"special addr but not port", eps, args{ips[1]}, false, true},
		{"not special", eps, args{ips[2]}, false, false},
		{"blank", eps, args{IP{}}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.epm.IsSpecial(tt.args.ip); got != tt.wantIP {
				t.Errorf("Endpoints.IsSpecial() = %v, want %v", got, tt.wantIP)
			}
			if got := tt.epm.IsSpecialAddress(tt.args.ip.Address); got != tt.wantAddr {
				t.Errorf("Endpoints.IsSpecial() = %v, want %v", got, tt.wantAddr)
			}
		})
	}
}

func TestEndpoints_Connections(t *testing.T) {
	eps := NewEndpoints()
	ips := testIPs()

	if amt, dur := eps.Connections(ips[0]); amt > 0 || dur > 0 {
		t.Errorf("Empty endpoint has connections for %s: %d %v", ips[0], amt, dur)
	}

	eps.RemoveConnection(IP{})

	//          NOT 6, last is duplicate
	for i := 0; i < 5; i++ {
		for j := 0; j <= i; j++ { // add ips[i] i-times
			eps.AddConnection(ips[i])
		}
	}
	time.Sleep(time.Millisecond)
	for i := 0; i < 5; i++ {
		if amt, dur := eps.Connections(ips[i]); amt != uint(i+1) || dur == 0 || dur > time.Millisecond*5 {
			t.Errorf("Pass 1: %s. expected = %d, have %d. duration %v", ips[i], i+1, amt, dur)
		}
	}

	for i := 0; i < 5; i++ {
		// remove one
		eps.RemoveConnection(ips[i])
	}

	for i := 0; i < 5; i++ {
		if amt, dur := eps.Connections(ips[i]); amt != uint(i) || dur == 0 || dur > time.Millisecond*5 {
			t.Errorf("Pass 2: %s. expected = %d, have %d. duration %v", ips[i], i+1, amt, dur)
		}
	}

	for i := 0; i < 5; i++ {
		for j := 0; j <= i; j++ { // remove all + 1
			eps.RemoveConnection(ips[i])
		}
	}

	for i := 0; i < 5; i++ {
		if amt, dur := eps.Connections(ips[i]); amt != 0 || dur == 0 || dur > time.Millisecond*5 {
			t.Errorf("Pass 3: %s. expected = %d, have %d. duration %v", ips[i], 0, amt, dur)
		}
	}
}
