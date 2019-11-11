package p2p

import (
	"sync"
	"time"
)

type Measure struct {
	parcelsIn  uint64
	parcelsOut uint64
	bytesIn    uint64
	bytesOut   uint64
	dataMtx    sync.Mutex

	rateParcelOut float64
	rateParcelIn  float64
	rateBytesOut  float64
	rateBytesIn   float64
	rateMtx       sync.RWMutex

	rate time.Duration
}

func NewMeasure(rate time.Duration) *Measure {
	m := new(Measure)
	m.rate = rate
	go m.calculate()
	return m
}

func (m *Measure) GetRate() (float64, float64, float64, float64) {
	m.rateMtx.RLock()
	defer m.rateMtx.RUnlock()
	return m.rateParcelIn, m.rateParcelOut, m.rateBytesIn, m.rateBytesOut
}

func (m *Measure) calculate() {
	ticker := time.NewTicker(m.rate)
	sec := m.rate.Seconds()
	for range ticker.C {
		m.dataMtx.Lock()
		m.rateMtx.Lock()

		m.rateParcelOut = float64(m.parcelsOut) / sec
		m.rateBytesOut = float64(m.bytesOut) / sec
		m.rateParcelIn = float64(m.parcelsIn) / sec
		m.rateBytesIn = float64(m.bytesIn) / sec

		m.parcelsIn = 0
		m.parcelsOut = 0
		m.bytesIn = 0
		m.bytesOut = 0

		m.dataMtx.Unlock()
		m.rateMtx.Unlock()
	}
}

func (m *Measure) Send(size uint64) {
	m.dataMtx.Lock()
	m.parcelsOut++
	m.bytesOut += size
	m.dataMtx.Unlock()
}

func (m *Measure) Receive(size uint64) {
	m.dataMtx.Lock()
	m.parcelsIn++
	m.bytesIn += size
	m.dataMtx.Unlock()
}
