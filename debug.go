package p2p

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"text/tabwriter"
)

// this file is for debugging only and not included in the factom repo

// DebugMessage is temporary
func (n *Network) DebugMessage() (string, int) {
	s := n.controller.peers.Slice()

	sort.Slice(s, func(i, j int) bool {
		return s[i].Hash < s[j].Hash
	})

	buf := new(strings.Builder)

	fmt.Fprintf(buf, "\nONLINE: (%d/%d/%d)\n", len(s), n.conf.TargetPeers, n.conf.MaxPeers)

	tw := tabwriter.NewWriter(buf, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Hash\tMPS Down\tMPS Up\tBps Down\tBps up\tRatio\tDropped\t\n")
	fmt.Fprintf(tw, "----\t--------\t------\t--------\t------\t-----\t-------\t\n")

	count := len(s)
	for _, p := range s {
		m := p.GetMetrics()
		fmt.Fprintf(tw, "%s\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%d\t\n", p.Hash, m.MPSDown, m.MPSUp, m.BPSDown, m.BPSUp, m.SendFillRatio, m.Dropped)
	}
	tw.Flush()

	fmt.Fprint(buf, "\nBANS:\n")
	tw = tabwriter.NewWriter(buf, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Address\tExpires\t\n")
	fmt.Fprintf(tw, "-------\t-------\t\n")

	n.controller.banMtx.RLock()
	for ip, ts := range n.controller.bans {
		fmt.Fprintf(tw, "%s\t%s\t\n", ip, ts.Format("2006-01-02 15:04:05"))
	}
	n.controller.banMtx.RUnlock()
	tw.Flush()

	return buf.String(), count
}

func (n *Network) DebugHalfviz() string {
	halfviz := ""

	idFromHash := func(s string) int64 {
		bits := strings.Split(s, " ")
		if len(bits) != 2 || len(bits[1]) != 8 {
			return -1
		}
		dec, err := hex.DecodeString(bits[1])
		if err != nil {
			return -2
		}
		return int64(binary.LittleEndian.Uint32(dec))
	}

	peers := n.controller.peers.Slice()
	for _, p := range peers {

		edge := ""
		id := idFromHash(p.Hash)
		if id < 0 {
			return fmt.Sprintf("[halfviz] unable to get id from %s (code %d)", p.Hash, id)
		}
		if n.conf.NodeID < 4 || id < 4 {
			min := n.conf.NodeID
			if uint32(id) < min {
				min = uint32(id)
			}
			if min != 0 {
				color := []string{"red", "green", "blue"}[min-1]
				edge = fmt.Sprintf(" {color:%s, weight=3}", color)
			}
		}

		if p.IsIncoming {
			halfviz += fmt.Sprintf("%s -> %s:%s%s\n", p.Endpoint, n.conf.BindIP, n.conf.ListenPort, edge)
		} else {
			halfviz += fmt.Sprintf("%s:%s -> %s%s\n", n.conf.BindIP, n.conf.ListenPort, p.Endpoint, edge)
		}
	}

	return halfviz
}

// DebugServer is temporary
func DebugServer(n *Network) {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug", func(rw http.ResponseWriter, req *http.Request) {
		a, _ := n.DebugMessage()
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
			out += fmt.Sprintf("\t\tCapacity: %.2f\n", m.SendFillRatio)
			out += fmt.Sprintf("\t\tDropped: %d\n", m.Dropped)
		}

		rw.Write([]byte(out))
	})

	go http.ListenAndServe("localhost:8070", mux)
}
