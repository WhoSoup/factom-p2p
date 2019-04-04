package p2p

var pcLogger = packageLogger.WithField("subpack", "protocol")

// ParcelChannel is a channel that supports non-blocking sends
type ParcelChannel chan *Parcel

func NewParcelChannel(capacity uint) ParcelChannel {
	return make(ParcelChannel, capacity)
}

func (pc ParcelChannel) Send(parcel *Parcel) {
	select { // hits default if sending message would block.
	case pc <- parcel:
	default:
		pcLogger.Warnf("ParcelChannel.Send() - Channel is full!")
	}
}

func (pc ParcelChannel) Reader() <-chan *Parcel {
	return pc
}
