package p2p

import (
	"reflect"
	"testing"
)

func Test_controller_parseSpecial(t *testing.T) {
	valid := []IP{
		{"127.0.0.1", "80", 2130706433},
		{"127.0.0.2", "8080", 2130706434},
		{"1.1.1.1", "8110", 16843009},
	}

	c := new(controller)
	c.logger = controllerLogger
	type args struct {
		raw string
	}
	tests := []struct {
		name string
		c    *controller
		args args
		want []IP
	}{
		{"bunch of addresses", c, args{"127.0.0.1:80,127.0.0.1,127.0.0.2,127.0.0.2:8080,1.1.1.1:8110,1.1.1.1:8089;127.0.0.1:4000"}, valid},
		{"single address", c, args{"127.0.0.1:80"}, []IP{{"127.0.0.1", "80", 2130706433}}},
		{"single bad address", c, args{"0.0.0.256:50"}, nil},
		{"single bad address 2", c, args{":50"}, nil},
		{"blank", c, args{""}, nil},
		{"just address", c, args{"127.0.0.1"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.parseSpecial(tt.args.raw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("controller.parseSpecial() = %v, want %v", got, tt.want)
			}
		})
	}
}
