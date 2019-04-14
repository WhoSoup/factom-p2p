package p2p

import "testing"

func Test_sorting(t *testing.T) {

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
