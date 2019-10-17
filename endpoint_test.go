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
		want    Endpoint
		wantErr bool
	}{
		{"ok localhost", args{"127.0.0.1", "8088"}, Endpoint{"127.0.0.1", "8088"}, false},
		{"ok other ip", args{"1.2.3.4", "8088"}, Endpoint{"1.2.3.4", "8088"}, false},
		{"empty", args{"", ""}, Endpoint{}, true},
		{"no port", args{"127.0.0.1", ""}, Endpoint{}, true},
		{"no ip", args{"", "8088"}, Endpoint{}, true},
		{"invalid ip", args{"127.0.0.256", "8088"}, Endpoint{}, true},
		{"domain lookup", args{"localhost", "8088"}, Endpoint{}, true}, // likely uses ::1 ipv6 address
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEndpoint(tt.args.addr, tt.args.port)
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
		want    Endpoint
		wantErr bool
	}{
		// valid formats use NewIP() which is tested above, so these test cases don't need to cover them again
		// only checking ones that would fail the parsing
		{"ok localhost", args{"127.0.0.1:80"}, Endpoint{"127.0.0.1", "80"}, false},
		{"port out of range", args{"127.0.0.1:70000"}, Endpoint{}, true},
		{"no port", args{"127.0.0.1"}, Endpoint{}, true},
		{"empty", args{""}, Endpoint{}, true},
		{"no ip", args{":80"}, Endpoint{}, true},
		{"bad ip", args{"127.0:80"}, Endpoint{}, true},
		{"wrong format 1", args{"127.0.0.1,80"}, Endpoint{}, true},
		{"wrong format 2", args{"127.0.0.1:80 test"}, Endpoint{}, true},
		{"wrong format 3", args{"ip:127.0.0.1 port:80"}, Endpoint{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEndpoint(tt.args.s)
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
