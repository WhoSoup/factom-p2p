package p2p

import "runtime"

var pcLogger = packageLogger.WithField("subpack", "protocol")

// ParcelChannel is a channel that supports non-blocking sends
type ParcelChannel chan *Parcel

func NewParcelChannel(capacity uint) ParcelChannel {
	return make(ParcelChannel, capacity)
}

func (pc ParcelChannel) Send(parcel *Parcel) {
	select {
	case pc <- parcel:
	default:
		_, f, l, _ := runtime.Caller(1)
		pcLogger.Warnf("ParcelChannel.Send() - Channel is full! Unable to deliver %s (%s:%d)", parcel, f, l)
	}
}

func (pc ParcelChannel) Reader() <-chan *Parcel {
	return pc
}
