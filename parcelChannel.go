package p2p

var pcLogger = packageLogger.WithField("subpack", "protocol")

// ParcelChannel is a channel that supports non-blocking sends
type ParcelChannel chan *Parcel

func newParcelChannel(capacity uint) ParcelChannel {
	return make(ParcelChannel, capacity)
}

// Send a parcel along this channel. Non-blocking. If full, half of messages are dropped.
func (pc ParcelChannel) Send(parcel *Parcel) bool {
	select {
	case pc <- parcel:
		return true
	default:
		dropped := 0
		for len(pc) > cap(pc)/2 {
			<-pc
			dropped++
		}
		pcLogger.Warnf("ParcelChannel.Send() - Channel is full! Dropped %d old messages", dropped)
		select {
		case pc <- parcel:
			return true
		default:
			return false
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
