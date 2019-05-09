package util

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Dialer is a construct to throttle dialing and limit by attempts
type Dialer struct {
	bindip      string
	interval    time.Duration
	timeout     time.Duration
	maxattempts uint
	attempts    map[IP]attempt
	attemptsMtx sync.RWMutex
}

type attempt struct {
	t time.Time
	c uint
}

// NewDialer creates a new Dialer
func NewDialer(ip string, interval, timeout, reset time.Duration, maxattempts uint) *Dialer {
	d := new(Dialer)
	d.bindip = ip
	d.interval = interval
	d.timeout = timeout
	d.maxattempts = maxattempts
	d.attempts = make(map[IP]attempt)
	return d
}

// CanDial checks if the given ip can be dialed yet
func (d *Dialer) CanDial(ip IP) bool {
	d.attemptsMtx.RLock()
	defer d.attemptsMtx.RUnlock()
	a, ok := d.attempts[ip]
	if !ok {
		return true
	}

	if time.Since(a.t) < d.interval {
		return false
	}

	if a.c >= d.maxattempts {
		return false
	}

	return true
}

// Dial an ip. Returns the active TCP connection or error if it failed to connect
func (d *Dialer) Dial(ip IP) (net.Conn, error) {
	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", d.bindip))
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{
		LocalAddr: local,
		Timeout:   d.timeout,
	}

	d.attemptsMtx.Lock()
	a, ok := d.attempts[ip]
	if ok {
		a.c++
		a.t = time.Now()
	} else {
		a = attempt{time.Now(), 1}
	}
	d.attempts[ip] = a
	d.attemptsMtx.Unlock()

	con, err := dialer.Dial("tcp", ip.String())
	if err != nil {
		return nil, err
	}
	return con, nil
}

// Reset an ip's attempt count and interval
func (d *Dialer) Reset(ip IP) {
	d.attemptsMtx.Lock()
	defer d.attemptsMtx.Unlock()
	a, ok := d.attempts[ip]
	if ok {
		a.c = 0
		d.attempts[ip] = a
	}
}

func (d *Dialer) Failed(ip IP) bool {
	d.attemptsMtx.RLock()
	defer d.attemptsMtx.RUnlock()
	return d.attempts[ip].c >= d.maxattempts
}
