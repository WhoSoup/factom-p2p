package p2p

import (
	"time"
)

// Configuration defines the behavior of the gossip network protocol
type Configuration struct {
	// Network is the NetworkID of the network to use, eg. MainNet, TestNet, etc
	Network NetworkID

	// NodeID is this node's id
	NodeID uint64
	// NodeName is the internal name of the node
	NodeName string

	// === Peer Management Settings ===
	// GossipConfiguration holds the configuration for the peer management system
	// PeerSaveInterval dictates how often peers should be saved to disk
	PeerSaveInterval time.Duration
	// PeerRequestInterval dictates how often neighbors should be asked for an
	// updated peer list
	PeerRequestInterval time.Duration
	// PeerReseedInterval dictates how often the seed file should be accessed
	// to check for changes
	PeerReseedInterval time.Duration
	// PeerIPLimit specifies the maximum amount of peers to accept from a single
	// ip address
	// 0 for unlimited
	PeerIPLimit uint

	// === Gossip Behavior ===
	// Outgoing is the number of peers this node attempts to connect to
	Outgoing uint
	// Incoming is the number of incoming connections this node is willing to accept
	Incoming uint
	// PeerShareAmount is the number of peers we share
	PeerShareAmount     uint
	MinimumQualityScore int32

	// Fanout controls how many random peers are selected for propagating messages
	// Higher values increase fault tolerance but also increase network congestion
	Fanout uint

	// TrustedOnly indicates whether or not to connect only to trusted peers
	TrustedOnly bool
	// RefuseIncoming indicates whether or not to disallow all incoming connections
	// for example if you are behind a NAT
	RefuseIncoming bool
	// Refuse unknown peers from connecting but allow special peers
	RefuseUnknown bool
	// PeersFile is the path to the file where peers will be stored
	PeersFile string
	// SeedURL is the URL of the remote seed file
	SeedURL string // URL to a source of peer info

	// ConfigPeers is a list of persistent peers from the factomd.conf file
	ConfigPeers string
	// CmdLinePeers is a list of peers passed via command line parameter
	CmdLinePeers string

	// === Connection Settings ===

	// ListenIP is the ip address to bind to when listening for incoming connections
	// leave blank to bind to all
	ListenIP string
	// ListenPort is the port to listen to incoming tcp connections on
	ListenPort string
	// ListenLimit is the lockout period of accepting connections from a single
	// ip after having a successful connection from that ip
	ListenLimit time.Duration

	// PingInterval dictates the maximum amount of time a connection can be
	// silent (no writes) before sending a Ping
	PingInterval time.Duration

	// RedialInterval dictates how long to wait between connection attempts
	RedialInterval time.Duration
	// RedialAttempts is the number of redial attempts to make before considering
	// a connection unreachable
	RedialAttempts uint

	// ReadDeadline is the maximum acceptable time to read a single parcel
	// if a connection takes longer, it is disconnected
	ReadDeadline time.Duration

	// WriteDeadline is the maximum acceptable time to send a single parcel
	// if a connection takes longer, it is disconnected
	WriteDeadline time.Duration

	ProtocolVersion uint16
	// ProtocolVersionMinimum is the earliest version this package supports
	ProtocolVersionMinimum uint16

	LogPath  string // Path for logs
	LogLevel string // Logging level
}

// DefaultP2PConfiguration returns a network configuration with base values
// These should be overwritten with command line and config parameters
func DefaultP2PConfiguration() *Configuration {
	c := new(Configuration)
	c.Network = MainNet
	c.NodeID = 0
	c.NodeName = "FNode0"

	c.PeerSaveInterval = time.Second * 30
	c.PeerRequestInterval = time.Minute * 3
	c.PeerReseedInterval = time.Hour * 4
	c.PeerIPLimit = 0

	c.Outgoing = 32
	c.Incoming = 150
	c.Fanout = 16
	c.PeerShareAmount = 4 * c.Outgoing // legacy math
	c.MinimumQualityScore = 20

	c.TrustedOnly = false
	c.RefuseIncoming = false

	c.ListenIP = "" // bind to all
	c.ListenPort = "8108"
	c.ListenLimit = time.Second
	c.PingInterval = time.Second * 15
	c.RedialInterval = time.Second * 20
	c.RedialAttempts = 5

	c.ReadDeadline = time.Minute * 5  // high enough to accomodate large packets
	c.WriteDeadline = time.Minute * 5 // but fail eventually

	c.ProtocolVersion = 9
	c.ProtocolVersionMinimum = 9

	return c
}
