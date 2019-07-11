package p2p

import (
	"reflect"
	"testing"
)

func TestNewIP(t *testing.T) {
	type args struct {
		addr string
		port string
	}
	tests := []struct {
		name    string
		args    args
		want    IP
		wantErr bool
	}{
		{"ok localhost", args{"127.0.0.1", "8088"}, IP{"127.0.0.1", "8088"}, false},
		{"ok other ip", args{"1.2.3.4", "8088"}, IP{"1.2.3.4", "8088"}, false},
		{"empty", args{"", ""}, IP{}, true},
		{"no port", args{"127.0.0.1", ""}, IP{}, true},
		{"no ip", args{"", "8088"}, IP{}, true},
		{"invalid ip", args{"127.0.0.256", "8088"}, IP{}, true},
		{"domain lookup", args{"localhost", "8088"}, IP{"localhost", "8088"}, false}, // likely uses ::1 ipv6 address
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIP(tt.args.addr, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAddress(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    IP
		wantErr bool
	}{
		// valid formats use NewIP() which is tested above, so these test cases don't need to cover them again
		// only checking ones that would fail the parsing
		{"ok localhost", args{"127.0.0.1:80"}, IP{"127.0.0.1", "80"}, false},
		{"port out of range", args{"127.0.0.1:70000"}, IP{}, true},
		{"no port", args{"127.0.0.1"}, IP{}, true},
		{"empty", args{""}, IP{}, true},
		{"no ip", args{":80"}, IP{}, true},
		{"bad ip", args{"127.0:80"}, IP{}, true},
		{"wrong format 1", args{"127.0.0.1,80"}, IP{}, true},
		{"wrong format 2", args{"127.0.0.1:80 test"}, IP{}, true},
		{"wrong format 3", args{"ip:127.0.0.1 port:80"}, IP{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAddress(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
