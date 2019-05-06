package p2p

// ParcelType is a list of parcel types that this node understands
type ParcelType uint16

const ( // iota is reset to 0
	TypeHeartbeat    ParcelType = iota // "Note, I'm still alive"
	TypePing                           // "Are you there?"
	TypePong                           // "yes, I'm here"
	TypePeerRequest                    // "Please share some peers"
	TypePeerResponse                   // "Here's some peers I know about."
	TypeAlert                          // network wide alerts (used in bitcoin to indicate criticalities)
	TypeMessage                        // Application level message
	TypeMessagePart                    // Application level message that was split into multiple parts
	TypeHandshake
)

var typeStrings = map[ParcelType]string{
	TypeHeartbeat:    "Heartbeat",     // "Note, I'm still alive"
	TypePing:         "Ping",          // "Are you there?"
	TypePong:         "Pong",          // "yes, I'm here"
	TypePeerRequest:  "Peer-Request",  // "Please share some peers"
	TypePeerResponse: "Peer-Response", // "Here's some peers I know about."
	TypeAlert:        "Alert",         // network wide alerts (used in bitcoin to indicate criticalities)
	TypeMessage:      "Message",       // Application level message
	TypeMessagePart:  "MessagePart",   // Application level message that was split into multiple parts
}

func (t ParcelType) String() string {
	return typeStrings[t]
}
