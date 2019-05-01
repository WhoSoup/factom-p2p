package p2p

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus struct {
	Connections prometheus.Gauge // done
	Unique      prometheus.Gauge
	Connecting  prometheus.Gauge // done
	Incoming    prometheus.Gauge // done
	Outgoing    prometheus.Gauge // done

	KnownPeers prometheus.Gauge // done

	SendRoutines    prometheus.Gauge
	ReceiveRoutines prometheus.Gauge

	ParcelsSent     prometheus.Counter
	ParcelsReceived prometheus.Counter
	Invalid         prometheus.Counter
	AppSent         prometheus.Counter
	AppReceived     prometheus.Counter

	once sync.Once
}

func (p *Prometheus) Setup() {
	p.once.Do(func() {

		ng := func(name, help string) prometheus.Gauge {
			g := prometheus.NewGauge(prometheus.GaugeOpts{
				Name: name,
				Help: help,
			})
			prometheus.MustRegister(g)
			return g
		}
		p.Connections = ng("factomd_p2p_peers_online", "Number of established connections")
		p.Unique = ng("factomd_p2p_peers_unique", "Number of unique ip addresses connected")
		p.Connecting = ng("factomd_p2p_peers_connecting", "Number of connections currently dialing or awaiting handshake")
		p.Incoming = ng("factomd_p2p_peers_incoming", "Number of peers that have dialed to this node")
		p.Outgoing = ng("factomd_p2p_peers_outgoing", "Number of peers that this node has dialed to")
		p.KnownPeers = ng("factomd_p2p_peers_known", "Number of peers known to the system")
		p.SendRoutines = ng("factomd_p2p_tech_sendroutines", "Number of active send routines")
		p.ReceiveRoutines = ng("factomd_p2p_tech_receiveroutines", "Number of active receive routines")
		p.ParcelsSent = ng("factomd_p2p_parcels_sent", "Total number of parcels sent out")
		p.ParcelsReceived = ng("factomd_p2p_parcels_received", "Total number of parcels received")
		p.Invalid = ng("factom_p2p_parcels_invalid", "Total number of invalid parcels received")
		p.AppSent = ng("factom_p2p_messages_sent", "Total number of application messages sent")
		p.AppReceived = ng("factom_p2p_messages_received", "Total number of application messages received")

	})
}
