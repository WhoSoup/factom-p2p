package util

import (
	"fmt"
	"testing"
	"time"
)

func TestFilter_Check(t *testing.T) {
	f := NewFilter(time.Millisecond*5, time.Millisecond)

	tests := []struct {
		name string
		f    *Filter
		hash string
	}{
		{"1-%d", f, "hash1"},
		{"2-%d", f, "hash2"},
		{"3-%d", f, "hash3"},
		{"4-%d", f, "hash4"},
		{"5-%d", f, "hash5"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf(tt.name, 1), func(t *testing.T) {
			if got := tt.f.Check(tt.hash); got != true {
				t.Errorf("Filter.Check() = %v, want %v", got, true)
			}
		})
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf(tt.name, 2), func(t *testing.T) {
			if got := tt.f.Check(tt.hash); got != false {
				t.Errorf("Filter.Check() = %v, want %v", got, false)
			}
		})
	}
	time.Sleep(time.Millisecond * 4)
	f.Check("foo")
	time.Sleep(time.Millisecond * 3)
	for _, tt := range tests {
		t.Run(fmt.Sprintf(tt.name, 3), func(t *testing.T) {
			if got := tt.f.Check(tt.hash); got != true {
				t.Errorf("Filter.Check() = %v, want %v", got, true)
			}
		})
	}
	if f.Check("foo") {
		t.Error("Final check is not in filter ")
	}
	f.Stop()
}
