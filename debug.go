package p2p

import (
	"fmt"
	"net/http"
	"sort"
)

// this file is for debugging only and not included in the factom repo

// DebugMessage is temporary
func (n *Network) DebugMessage() (string, string, int) {
	hv := ""
	s := n.controller.peers.Slice()
	r := fmt.Sprintf("\nONLINE: (%d/%d/%d)\n", len(s), n.conf.TargetPeers, n.conf.MaxPeers)
	count := len(s)
	for _, p := range s {

		metrics := p.GetMetrics()
		r += fmt.Sprintf("\tPeer %s (MPS %.2f/%.2f) (BPS %.2f/%.2f) (Cap %.2f)\n", p.String(), metrics.MPSDown, metrics.MPSUp, metrics.BPSDown, metrics.BPSUp, metrics.SendUsageRate)
		edge := ""
		if n.conf.NodeID < 4 || p.NodeID < 4 {
			min := n.conf.NodeID
			if p.NodeID < min {
				min = p.NodeID
			}
			if min != 0 {
				color := []string{"red", "green", "blue"}[min-1]
				edge = fmt.Sprintf(" {color:%s, weight=3}", color)
			}
		}
		if p.IsIncoming {
			hv += fmt.Sprintf("%s -> %s:%s%s\n", p.Endpoint, n.conf.BindIP, n.conf.ListenPort, edge)
		} else {
			hv += fmt.Sprintf("%s:%s -> %s%s\n", n.conf.BindIP, n.conf.ListenPort, p.Endpoint, edge)
		}
	}
	known := ""
	/*for _, ip := range n.controller.endpoints.IPs() {
		known += ip.Address + " "
	}*/
	r += "\nKNOWN:\n" + known

	/*banned := ""
	for ip, time := range n.controller.endpoints.Bans {
		banned += fmt.Sprintf("\t%s %s\n", ip, time)
	}
	r += "\nBANNED:\n" + banned*/
	return r, hv, count
}

// DebugServer is temporary
func DebugServer(n *Network) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		a, _, _ := n.DebugMessage()
		rw.Write([]byte(a))
	})

	mux.HandleFunc("/stats", func(rw http.ResponseWriter, req *http.Request) {
		out := ""
		out += fmt.Sprintf("Channels\n")
		out += fmt.Sprintf("\tToNetwork: %d / %d\n", len(n.ToNetwork), cap(n.ToNetwork))
		out += fmt.Sprintf("\tFromNetwork: %d / %d\n", len(n.FromNetwork), cap(n.FromNetwork))
		out += fmt.Sprintf("\tpeerData: %d / %d\n", len(n.controller.peerData), cap(n.controller.peerData))
		out += fmt.Sprintf("\nPeers (%d)\n", n.controller.peers.Total())

		slice := n.controller.peers.Slice()
		sort.Slice(slice, func(i, j int) bool {
			return slice[i].connected.Before(slice[j].connected)
		})

		for _, p := range slice {
			out += fmt.Sprintf("\t%s\n", p.Endpoint)
			out += fmt.Sprintf("\t\tsend: %d / %d\n", len(p.send), cap(p.send))
			m := p.GetMetrics()
			out += fmt.Sprintf("\t\tBytesReceived: %d\n", m.BytesReceived)
			out += fmt.Sprintf("\t\tBytesSent: %d\n", m.BytesSent)
			out += fmt.Sprintf("\t\tMessagesSent: %d\n", m.MessagesSent)
			out += fmt.Sprintf("\t\tMessagesReceived: %d\n", m.MessagesReceived)
			out += fmt.Sprintf("\t\tMomentConnected: %s\n", m.MomentConnected)
			out += fmt.Sprintf("\t\tPeerQuality: %d\n", m.PeerQuality)
			out += fmt.Sprintf("\t\tIncoming: %v\n", m.Incoming)
			out += fmt.Sprintf("\t\tLastReceive: %s\n", m.LastReceive)
			out += fmt.Sprintf("\t\tLastSend: %s\n", m.LastSend)
			out += fmt.Sprintf("\t\tMPS Down: %.2f\n", m.MPSDown)
			out += fmt.Sprintf("\t\tMPS Up: %.2f\n", m.MPSUp)
			out += fmt.Sprintf("\t\tBPS Down: %.2f\n", m.BPSDown)
			out += fmt.Sprintf("\t\tBPS Up: %.2f\n", m.BPSUp)
			out += fmt.Sprintf("\t\tCapacity: %.2f\n", m.SendUsageRate)
			out += fmt.Sprintf("\t\tDropped: %d\n", m.Dropped)
		}

		rw.Write([]byte(out))
	})

	go http.ListenAndServe("localhost:8070", mux)
}
