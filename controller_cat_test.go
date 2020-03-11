package p2p

import (
	"reflect"
	"testing"
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
