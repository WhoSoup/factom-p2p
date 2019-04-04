package p2p

import "sync"

var parcelFactoryOnce sync.Once
var parcelFactory *ParcelFactory

type ParcelFactory struct {
	config *Configuration
}

func NewParcelFactory() *ParcelFactory {
	parcelFactoryOnce.Do(func() {
		parcelFactory = new(ParcelFactory)
	})
	return parcelFactory
}

func (pf *ParcelFactory) Init(config *Configuration) {
	pf.config = config
}

func NewParcel(command ParcelCommandType, payload []byte) *Parcel {
	factory := NewParcelFactory()

	header := new(ParcelHeader)
	header.Network = factory.config.Network
	header.Version = factory.config.ProtocolVersion
	header.Type = TypeMessage
	header.TargetPeer = "" // initially no target
	header.PeerPort = ""   // store our listening port

	header.AppHash = "NetworkMessage"
	header.AppType = "Network"
	parcel := new(Parcel).Init(*header)
	parcel.Payload = payload
	parcel.Header.Type = command
	parcel.UpdateHeader() // Updates the header with info about payload.
	return parcel
}

func (p *ParcelHeader) Init(network NetworkID) *ParcelHeader {

	return p
}

func (p *Parcel) Init(header ParcelHeader) *Parcel {
	p.Header = header
	return p
}
