package p2p

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

const testNetworkPeers = 6

func testStartNetwork(id int) (*Network, error) {
	conf := DefaultP2PConfiguration()
	conf.Network = 1
	conf.DialTimeout = time.Millisecond * 10
	conf.HandshakeTimeout = time.Millisecond * 50
	conf.RedialInterval = time.Millisecond * 25
	conf.ListenLimit = time.Millisecond * 20
	conf.MinReseed = testNetworkPeers
	conf.PeerRequestInterval = time.Millisecond * 10
	conf.NodeName = fmt.Sprintf("TestNode%d", id)
	conf.PeerCacheFile = ""
	conf.BindIP = "127.0.0.1"
	conf.TargetPeers = testNetworkPeers
	conf.MaxPeers = testNetworkPeers + 2
	conf.DropTo = testNetworkPeers - 1
	conf.SeedURL = fmt.Sprintf("http://127.0.0.1:14120/seed%d", id)
	conf.ListenPort = fmt.Sprintf("%d", 14121+id)
	conf.EnablePrometheus = false
	conf.ProtocolVersion = uint16(9 + id%3)
	n, err := NewNetwork(conf)
	if err != nil {
		return nil, err
	}
	n.Run()
	return n, nil
}

func TestNetwork_Run(t *testing.T) {
	lvl := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	old := loopTimer
	loopTimer = time.Millisecond * 5
	defer func() {
		loopTimer = old
		logrus.SetLevel(lvl)
	}()

	mux := http.NewServeMux()
	for i := 0; i < testNetworkPeers; i++ {
		mux.HandleFunc(fmt.Sprintf("/seed%d", i), func(rw http.ResponseWriter, req *http.Request) {
			for j := 0; j < i; j++ {
				rw.Write([]byte(fmt.Sprintf("127.0.0.1:%d\n", 14121+j)))
			}
		})
	}

	srv := &http.Server{Addr: "127.0.0.1:14120", Handler: mux}
	go srv.ListenAndServe()
	defer srv.Close()

	networks := make([]*Network, 0, testNetworkPeers)
	for i := 0; i < testNetworkPeers; i++ {
		if n, err := testStartNetwork(i); err != nil {
			t.Errorf("can't start network %d: %v", i, err)
		} else {
			networks = append(networks, n)
		}
	}

	time.Sleep(time.Second * 5)

	for _, n := range networks {
		//t.Error(n.controller.peers.Slice())
		n.Stop()
	}
}
