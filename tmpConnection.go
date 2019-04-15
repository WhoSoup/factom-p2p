package p2p

import (
	"encoding/gob"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

type TmpConnection struct {
	conn net.Conn

	Send       ParcelChannel // messages from the other side
	Receive    ParcelChannel // messages to the other side
	Error      chan error    // connection died
	IsOutgoing bool

	writeDeadline time.Duration
	readDeadline  time.Duration
	encoder       *gob.Encoder
	decoder       *gob.Decoder

	// logging
	logger *log.Entry
}
