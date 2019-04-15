package p2p

import "net"

type TmpPool struct {
	stop    chan bool
	net     *Network
	receive ParcelChannel

	connections map[string]*Connection
}

func NewTmpPool(net *Network) *TmpPool {
	tpl := new(TmpPool)
	tpl.net = net
	tpl.receive = NewParcelChannel(net.conf.ChannelCapacity)
	tpl.stop = make(chan bool, 1)
	return tpl
}

func (tp *TmpPool) Start() {
	go tp.listen()
}

func (tp *TmpPool) Stop() {
	tp.stop <- true
}

func (tp *TmpPool) CreateIncoming(con net.Conn) {
	//c := NewConnection(tp.net, con, tp.receive)

}

func (tp *TmpPool) listen() {
	for {
		select {
		case <-tp.stop:
			return
		case <-tp.receive:

		}
	}
}
