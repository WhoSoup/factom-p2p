package p2p

import (
	"reflect"
	"testing"
	"time"
)

func Test_controller_processPeerShare(t *testing.T) {
	net := testNetworkHarness(t)

	share := make([]Endpoint, 3)
	for i := range share {
		share[i] = testRandomEndpoint()
	}

	v9p := new(Peer)
	v9p.prot = newProtocolV9(net.conf.Network, net.conf.NodeID, net.conf.ListenPort, nil, nil)

	v10p := new(Peer)
	v10p.prot = newProtocolV10(nil, nil)

	v11p := new(Peer)
	v11p.prot = newProtocolV11(nil)

	tests := []struct {
		name string
		peer *Peer
		want []Endpoint
	}{
		{"v9", v9p, share},
		{"v10", v10p, share},
		{"v11", v11p, share},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := tt.peer.prot.MakePeerShare(share)
			if err != nil {
				t.Errorf("prot.MakePeerShare() = %v", err)
			}

			if len(payload) == 0 {
				t.Fatal("can't proceed with empty payload")
			}

			parcel := newParcel(TypePeerResponse, payload)

			if got := net.controller.processPeerShare(tt.peer, parcel); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("controller.processPeerShare() = %v, want %v", got, tt.want)
			}

			payload[0] += 1
			parcelBad := newParcel(TypePeerResponse, payload)

			if got := net.controller.processPeerShare(tt.peer, parcelBad); reflect.DeepEqual(got, tt.want) {
				t.Errorf("controller.processPeerShare() equal even though parcel changed")
			}
		})
	}
}

func Test_controller_shuffleTrimShare(t *testing.T) {
	net := testNetworkHarness(t)

	eps := make([]Endpoint, 64)
	for i := range eps {
		eps[i] = testRandomEndpoint()
	}

	net.conf.PeerShareAmount = 64

	got := net.controller.shuffleTrimShare(eps)

	if uint(len(got)) != net.conf.PeerShareAmount {
		t.Errorf("shuffleTrimShare() trimmed when it shouldn't have. len = %d, want = %d", len(got), net.conf.PeerShareAmount)
	} else {
		eq := true
		for i, ep := range eps {
			if !got[i].Equal(ep) {
				eq = false
				break
			}
		}
		if eq {
			t.Errorf("shuffleTrimShare() did not shuffle. order matched")
		}
	}

	net.conf.PeerShareAmount = 32

	got = net.controller.shuffleTrimShare(eps)
	if uint(len(got)) != net.conf.PeerShareAmount {
		t.Errorf("shuffleTrimShare() did not trim. len = %d, want = %d", len(got), net.conf.PeerShareAmount)
	}

}

func Test_controller_sharePeers(t *testing.T) {
	net := testNetworkHarness(t)
	p := testRandomPeer(net)
	p._setProtocol(10, nil) // only need to encode peer share, not send on wire

	share := make([]Endpoint, 128)
	for i := range share {
		share[i] = testRandomEndpoint()
	}

	net.controller.sharePeers(p, share)

	if len(p.send) == 0 {
		t.Error("peer did not receive a parcel")
	} else {
		parc := <-p.send

		list, err := p.prot.ParsePeerShare(parc.Payload)
		if err != nil {
			t.Fatalf("error unmarshalling peer share %v", err)
		}

		if !testEqualEndpointList(share, list) {
			t.Errorf("endpoint lists did not match")
		}
	}
}

func Test_controller_asyncPeerRequest(t *testing.T) {
	net := testNetworkHarness(t)
	peer := testRandomPeer(net)

	peer._setProtocol(10, nil) // no connection needed

	sendShare := make([]Endpoint, net.conf.PeerShareAmount)
	for i := range sendShare {
		sendShare[i] = testRandomEndpoint()
	}

	done := make(chan bool)
	go func() {
		share, err := net.controller.asyncPeerRequest(peer)
		if err != nil {
			t.Errorf("async error %v", err)
		} else if !testEqualEndpointList(share, sendShare) {
			t.Errorf("async received different share")
		}
		done <- true
	}()

	time.Sleep(time.Millisecond * 100)

	net.controller.shareMtx.RLock()
	async, ok := net.controller.shareListener[peer.Hash]
	net.controller.shareMtx.RUnlock()

	if !ok {
		t.Fatal("no async channel found for peer")
	}

	if len(peer.send) == 0 {
		t.Errorf("peer did not send a request")
	} else if parc := <-peer.send; parc.ptype != TypePeerRequest {
		t.Errorf("parcel sent was not a request for peers. got = %s", parc.ptype)
	}

	payload, _ := peer.prot.MakePeerShare(sendShare)

	parc := newParcel(TypePeerResponse, payload)
	async <- parc

	<-done
}

func Test_controller_asyncPeerRequest_timeout(t *testing.T) {
	net := testNetworkHarness(t)
	net.conf.PeerShareTimeout = time.Millisecond * 50
	peer := testRandomPeer(net)

	got, err := net.controller.asyncPeerRequest(peer)
	if err == nil {
		t.Errorf("async did not return a timeout error. got = %v", got)
	}
}
