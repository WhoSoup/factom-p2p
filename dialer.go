package p2p

import (
	"fmt"
	"net"
	"time"
)

type Dialer struct {
	net      *Network
	attempts map[string]attempt
}

type attempt struct {
	t time.Time
	c uint
}

func NewDialer(n *Network) *Dialer {
	d := new(Dialer)
	d.net = n
	d.attempts = make(map[string]attempt)
	return d
}

func (d *Dialer) CanDial(ip IP) bool {
	a, ok := d.attempts[ip.String()]
	if !ok {
		return true
	}

	if time.Since(a.t) < d.net.conf.RedialInterval {
		return false
	}

	if a.c >= d.net.conf.RedialAttempts {
		return false
	}

	return true
}

func (d *Dialer) Dial(ip IP) (net.Conn, error) {
	local, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", d.net.conf.BindIP))
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{
		LocalAddr: local,
		Timeout:   d.net.conf.DialTimeout,
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
