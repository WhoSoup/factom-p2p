package util

import (
	"sync"
	"time"
)

type Filter struct {
	expiration time.Duration
	cleanTimer time.Duration
	hashes     map[string]time.Time
	tRecords   []item
	mtx        sync.Mutex
	stop       chan bool
}

type item struct {
	h string
	t time.Time
}

func NewFilter(t time.Duration, ct time.Duration) *Filter {
	f := new(Filter)
	f.expiration = t
	f.cleanTimer = ct
	f.hashes = make(map[string]time.Time)
	f.stop = make(chan bool, 1)
	go f.clean()
	return f
}

// Check returns TRUE if the hash is new, false if the hash is a duplicate
func (f *Filter) Check(hash string) bool {
	n := time.Now()
	cutoff := n.Add(-f.expiration)
	f.mtx.Lock()
	if !f.hashes[hash].Before(cutoff) {
		f.mtx.Unlock()
		return false
	}
	f.hashes[hash] = n
	f.tRecords = append(f.tRecords, item{hash, time.Now()})
	f.mtx.Unlock()
	return true
}

func (f *Filter) Stop() {
	f.stop <- true
}

func (f *Filter) clean() {
	for {
		select {
		case <-f.stop:
			return
		case <-time.After(f.cleanTimer):
			cutoff := time.Now().Add(-f.expiration)
			f.mtx.Lock()
			if len(f.tRecords) > 0 && f.tRecords[0].t.Before(cutoff) {
				i := 0
				for _, v := range f.tRecords {
					if cutoff.Before(v.t) {
						break
					}
					i++
					delete(f.hashes, v.h)
				}
				f.tRecords = f.tRecords[i:]

			}
			f.mtx.Unlock()
		}
	}
}
