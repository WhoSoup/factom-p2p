package p2p

import (
	"testing"
	"time"
)

func TestMeasure_calculate(t *testing.T) {
	tests := []struct {
		name  string
		m     *Measure
		want  float64
		want1 float64
		want2 float64
		want3 float64
	}{
		{"empty", &Measure{}, 0, 0, 0, 0},
		{"zero", &Measure{parcelsIn: 0, parcelsOut: 0, bytesIn: 0, bytesOut: 0}, 0, 0, 0, 0},
		{"zero over time", &Measure{parcelsIn: 0, parcelsOut: 0, bytesIn: 0, bytesOut: 0, rate: time.Second}, 0, 0, 0, 0},
		{"data without rate", &Measure{parcelsIn: 5, parcelsOut: 5, bytesIn: 5, bytesOut: 5}, 0, 0, 0, 0},
		{"one second", &Measure{parcelsIn: 1, parcelsOut: 2, bytesIn: 3, bytesOut: 4, rate: time.Second}, 1, 2, 3, 4},
		{"one minute", &Measure{parcelsIn: 60, parcelsOut: 120, bytesIn: 180, bytesOut: 240, rate: time.Minute}, 1, 2, 3, 4},
		{"fracs", &Measure{parcelsIn: 450, parcelsOut: 1800, bytesIn: 36, bytesOut: 3599, rate: time.Hour}, 0.125, .5, .01, 0.9997222222222222},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.calculate()
			got, got1, got2, got3 := tt.m.GetRate()
			if got != tt.want {
				t.Errorf("Measure.GetRate() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Measure.GetRate() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("Measure.GetRate() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("Measure.GetRate() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func TestMeasure_Send_Receive(t *testing.T) {
	m := &Measure{}
	if m.parcelsIn != 0 || m.parcelsOut != 0 || m.bytesIn != 0 || m.bytesOut != 0 {
		t.Errorf("measure initialized wrong: %d %d %d %d", m.parcelsIn, m.parcelsOut, m.bytesIn, m.bytesOut)
	}

	m.Send(123)
	m.Receive(321)

	if m.parcelsIn != 1 || m.parcelsOut != 1 || m.bytesIn != 321 || m.bytesOut != 123 {
		t.Errorf("case 1 wrong: %d %d %d %d", m.parcelsIn, m.parcelsOut, m.bytesIn, m.bytesOut)
	}

	m.Send(1000)
	m.Receive(2000)

	if m.parcelsIn != 2 || m.parcelsOut != 2 || m.bytesIn != 2321 || m.bytesOut != 1123 {
		t.Errorf("case 2 wrong: %d %d %d %d", m.parcelsIn, m.parcelsOut, m.bytesIn, m.bytesOut)
	}

}
