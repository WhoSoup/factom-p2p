package p2p

import "runtime"

var pcLogger = packageLogger.WithField("subpack", "protocol")

// ParcelChannel is a channel that supports non-blocking sends
type ParcelChannel chan *Parcel

func newParcelChannel(capacity uint) ParcelChannel {
	return make(ParcelChannel, capacity)
}

// Send a parcel along this channel. Non-blocking. If full, newer messages are dropped.
func (pc ParcelChannel) Send(parcel *Parcel) {
	select {
	case pc <- parcel:
	default:
		_, f, l, _ := runtime.Caller(1)
		pcLogger.Warnf("ParcelChannel.Send() - Channel is full! Unable to deliver %s (%s:%d)", parcel, f, l)
	}
}

// Reader returns a read-only channel
func (pc ParcelChannel) Reader() <-chan *Parcel {
	return pc
}
