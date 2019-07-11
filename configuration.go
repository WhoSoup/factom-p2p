package p2p

import (
	"time"
)

const (
	// Broadcast sends a parcel to multiple peers (randomly selected based on fanout and special peers)
	Broadcast = "<BROADCAST>"
	// FullBroadcast sends a parcel to all peers
	FullBroadcast = "<FULLBORADCAST>"
	// RandomPeer sends a parcel to one randomly selected peer
	RandomPeer = "<RANDOMPEER>"
)

// Configuration defines the behavior of the gossip network protocol
type Configuration struct {
	// Network is the NetworkID of the network to use, eg. MainNet, TestNet, etc
	Network NetworkID

	// NodeID is this node's id
	NodeID uint32
	// NodeName is the internal name of the node
	NodeName string

	// === Peer Management Settings ===
	// PeerRequestInterval dictates how often neighbors should be asked for an
	// updated peer list
	PeerRequestInterval time.Duration
	// PeerReseedInterval dictates how often the seed file should be accessed
	// to check for changes
	PeerReseedInterval time.Duration
	// PeerIPLimit specifies the maximum amount of peers to accept from a single
	// ip address
	// 0 for unlimited
	PeerIPLimitIncoming uint
	PeerIPLimitOutgoing uint

	// Special is a list of special peers, separated by comma. If no port is specified, the entire
	// ip is considered special
	Special string

	// PersistFile is the filepath to the file to save peers
	PersistFile string
	// how often to save these
	PersistInterval time.Duration
	PersistLevel    uint // 0 persist all peers
	// 1 persist peers we have had a connection with
	// 2 persist only peers we have been able to dial to
	PersistMinimum time.Duration // the minimum amount of time a connection has to last to last
	// PersistAgeLimit dictates how long a peer can be offline before being considered dead
	PersistAgeLimit time.Duration

	// to count as being connected
	// PeerShareAmount is the number of peers we share
	PeerShareAmount     uint
	MinimumQualityScore int32

	// CAT Settings
	RoundTime time.Duration
	Target    uint
	Max       uint
	Drop      uint
	MinReseed uint

	// === Gossip Behavior ===
	// Outgoing is the number of peers this node attempts to connect to
	Outgoing uint
	// Incoming is the number of incoming connections this node is willing to accept
	Incoming uint

	// Fanout controls how many random peers are selected for propagating messages
	// Higher values increase fault tolerance but also increase network congestion
	Fanout uint

	// SeedURL is the URL of the remote seed file
	SeedURL string // URL to a source of peer info

	// === Connection Settings ===

	// BindIP is the ip address to bind to for listening and connecting
	//
	// leave blank to bind to all
	BindIP string
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
	// RedialReset dictates after how long we should try to reconnect again
	RedialReset time.Duration
	// RedialAttempts is the number of redial attempts to make before considering
	// a connection unreachable
	RedialAttempts uint
	// DisconnectLock dictates how long the peer manager should wait for an incoming peer
	// to reconnect before considering dialing to them
	DisconnectLock time.Duration

	// ManualBan is the duration to ban an address for when banned manually
	ManualBan time.Duration
	// AutoBan is the duration to ban an address for when their qualityscore drops too low
	AutoBan time.Duration

	// HandshakeDeadline is the maximum acceptable time for an incoming conneciton
	// to send the first parcel after connecting
	HandshakeTimeout time.Duration
	DialTimeout      time.Duration

	// ReadDeadline is the maximum acceptable time to read a single parcel
	// if a connection takes longer, it is disconnected
	ReadDeadline time.Duration

	// WriteDeadline is the maximum acceptable time to send a single parcel
	// if a connection takes longer, it is disconnected
	WriteDeadline time.Duration

	// DuplicateFilter is how long message hashes are cashed to filter out duplicates.
	// Zero to disable
	DuplicateFilter time.Duration
	// DuplicateFilterCleanup is how frequently the cleanup mechanism is run to trim
	// memory of the duplicate filter
	DuplicateFilterCleanup time.Duration

	ProtocolVersion uint16
	// ProtocolVersionMinimum is the earliest version this package supports
	ProtocolVersionMinimum uint16

	ChannelCapacity uint

	LogPath  string // Path for logs
	LogLevel string // Logging level

	EnablePrometheus bool // Enable prometheus logging. Disable if you run multiple instances
}

// DefaultP2PConfiguration returns a network configuration with base values
// These should be overwritten with command line and config parameters
func DefaultP2PConfiguration() (c Configuration) {
	c.Network = MainNet
	c.NodeID = 0
	c.NodeName = "FNode0"

	c.PeerRequestInterval = time.Minute * 3
	c.PeerReseedInterval = time.Hour * 4
	c.PeerIPLimitIncoming = 0
	c.PeerIPLimitOutgoing = 0
	c.ManualBan = time.Hour * 24 * 7 // a week
	c.AutoBan = time.Hour * 24 * 7   // a week

	c.PersistFile = ""
	c.PersistInterval = time.Minute * 15

	c.Outgoing = 32
	c.Incoming = 150
	c.Fanout = 8
	c.PeerShareAmount = 4 * c.Outgoing // legacy math
	c.MinimumQualityScore = 20
	c.PersistLevel = 2
	c.PersistMinimum = time.Minute

	c.BindIP = "" // bind to all
	c.ListenPort = "8108"
	c.ListenLimit = time.Second
	c.PingInterval = time.Second * 15
	c.PersistAgeLimit = time.Hour * 48 // 2 days
	c.RedialInterval = time.Second * 20
	c.RedialReset = time.Hour * 12
	c.RedialAttempts = 5
	c.DisconnectLock = time.Minute * 3 // RedialInterval * RedialAttempts + 80 seconds

	c.ReadDeadline = time.Minute * 5      // high enough to accomodate large packets
	c.WriteDeadline = time.Minute * 5     // but fail eventually
	c.HandshakeTimeout = time.Second * 10 // can be quite low
	c.DialTimeout = time.Second * 10      // can be quite low

	c.DuplicateFilter = time.Hour
	c.DuplicateFilterCleanup = time.Minute

	c.ProtocolVersion = 10
	c.ProtocolVersionMinimum = 9

	c.ChannelCapacity = 5000

	c.EnablePrometheus = true

	return
}
