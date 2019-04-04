package p2p

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// LimitedListener will block multiple connection attempts from a single ip
// within a specific timeframe
type LimitedListener struct {
	net.Listener

	limit          time.Duration
	lastConnection time.Time
	history        []limitedConnect
}

type limitedConnect struct {
	address string
	time    time.Time
}

func NewLimitedListener(address string, limit time.Duration) (*LimitedListener, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	if limit < 0 {
		return nil, fmt.Errorf("Invalid time limit (negative)")
	}
	return &LimitedListener{
		Listener:       l,
		limit:          limit,
		lastConnection: time.Time{},
		history:        nil,
	}, nil
}

// clearHistory truncates the history to only relevant entries
func (ll *LimitedListener) clearHistory() {
	tl := time.Now().Add(-ll.limit) // get timelimit of range to check

	// no connection made in the last X seconds
	// the vast majority of connections will proc this
	if ll.lastConnection.Before(tl) {
		ll.history = nil // reset and release to gc
	}

	if len(ll.history) > 0 {
		i := 0
		for ; i < len(ll.history); i++ {
			if ll.history[i].time.After(tl) { // inside target range
				break
			}
		}

		if i >= len(ll.history) {
			ll.history = nil
		} else {
			ll.history = ll.history[i:]
		}
	}
}

// isInHistory checks if an address has connected in the last X seconds
// clears history before checking
func (ll *LimitedListener) isInHistory(addr string) bool {
	ll.clearHistory()

	for _, h := range ll.history {
		if h.address == addr {
			return true
		}
	}
	return false
}

// addToHistory adds an address to the system at the current time
func (ll *LimitedListener) addToHistory(addr string) {
	ll.history = append(ll.history, limitedConnect{address: addr, time: time.Now()})
	ll.lastConnection = time.Now()
}

// Accept accepts a connection if no other connection attempt from that ip has been made
// in the specified time frame
func (ll *LimitedListener) Accept() (net.Conn, error) {
	con, err := ll.Listener.Accept()
	if err != nil {
		return nil, err
	}

	addr := strings.Split(con.RemoteAddr().String(), ":")
	if ll.isInHistory(addr[0]) {
		con.Close()
		return nil, fmt.Errorf("connection rate limit exceeded")
	}

	ll.addToHistory(addr[0])
	return con, nil
}
