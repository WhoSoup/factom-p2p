package p2p

import "time"

type EndpointMap struct {
	net  *Network
	ends map[string]*Endpoint

	// TODO implement
	special map[string]bool
	banned  map[string]time.Time
}

func NewEndpointMap(n *Network) *EndpointMap {
	epm := new(EndpointMap)
	epm.net = n
	epm.ends = make(map[string]*Endpoint)
	return epm
}

func (epm *EndpointMap) Create(address, port string) (*Endpoint, error) {
	if e, ok := epm.ends[address]; ok {
		e.knownPorts[port] = true
		return e, nil
	}
	e, err := NewEndpoint(address)
	if err != nil {
		return nil, err
	}
	epm.ends[address] = e
	e.knownPorts[port] = true
	return e, nil
}

func (epm *EndpointMap) Get(address, port string) *Endpoint {
	if e, ok := epm.ends[address]; ok {
		if e.knownPorts[port] {
			return e
		}
	}
	return nil
}

func (epm *EndpointMap) Known(address, port string) bool {
	if e, ok := epm.ends[address]; ok {
		return e.knownPorts[port]
	}
	return false

}
