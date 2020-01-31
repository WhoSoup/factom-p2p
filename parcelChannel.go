// Copyright 2019-2020 Factomize LLC
// More information in the LICENSE file

package p2p

var pcLogger = packageLogger.WithField("subpack", "protocol")

// ParcelChannel is a channel that supports non-blocking sends
type ParcelChannel chan *Parcel

func newParcelChannel(capacity uint) ParcelChannel {
	return make(ParcelChannel, capacity)
}

// Send a parcel along this channel. Non-blocking. If full, half of messages are dropped.
func (pc ParcelChannel) Send(parcel *Parcel) (bool, int) {
	select {
	case pc <- parcel:
		return true, 0
	default:
		dropped := 0
		for len(pc) > cap(pc)/2 {
			<-pc
			dropped++
		}
		pcLogger.Warnf("ParcelChannel.Send() - Channel is full! Dropped %d old messages", dropped)
		select {
		case pc <- parcel:
			return true, dropped
		default:
			return false, dropped
		}
	}
}

// Reader returns a read-only channel
func (pc ParcelChannel) Reader() <-chan *Parcel {
	return pc
}

// Capacity returns a percentage [0.0,1.0] of how full the channel is
func (pc ParcelChannel) Capacity() float64 {
	return float64(len(pc)) / float64(cap(pc))
}
