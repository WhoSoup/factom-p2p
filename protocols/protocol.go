package protocols

import p2p "github.com/whosoup/factom-p2p"

type Protocol interface {
	PeerShareMarshal(ps []*p2p.Peer) []byte
	PeerShareUnmarshal(payload []byte) []p2p.PeerShare
}
