package p2p

import (
	"fmt"
	"net"
	"time"
)

type Dialer struct {
	bindip      string
	interval    time.Duration
	timeout     time.Duration
	maxattempts uint
	attempts    map[string]attempt
}

type attempt struct {
	t time.Time
	c uint
}

func NewDialer(ip string, interval, timeout time.Duration, maxattempts uint) *Dialer {
	d := new(Dialer)
	d.bindip = ip
	d.interval = interval
	d.timeout = timeout
	d.maxattempts = maxattempts
	d.attempts = make(map[string]attempt)
	return d
}

func (d *Dialer) CanDial(ip IP) bool {
	a, ok := d.attempts[ip.String()]
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

func (d *Dialer) Dial(ip IP) (net.Conn, error) {
	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", d.bindip))
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{
		LocalAddr: local,
		Timeout:   d.timeout,
	}

	a, ok := d.attempts[ip.String()]
	if ok {
		a.c++
		a.t = time.Now()
	} else {
		a = attempt{time.Now(), 1}
	}
	d.attempts[ip.String()] = a

	con, err := dialer.Dial("tcp", ip.String())
	if err != nil {
		return nil, err
	}
	return con, nil
}

func (d *Dialer) Reset(ip IP) {
	a, ok := d.attempts[ip.String()]
	if ok {
		a.c = 0
		d.attempts[ip.String()] = a
	}
}
