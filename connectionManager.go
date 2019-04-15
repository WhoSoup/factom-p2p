package p2p

import "sync"

type ConnectionManager struct {
	connsMutex sync.RWMutex
	net        *Network
	exists     map[string]bool
	conns      []*Connection
	byNodeID   map[string]map[uint64]*Connection

	manage chan *Connection
	stop   chan bool
}

func NewConnectionManager(n *Network) *ConnectionManager {
	cm := new(ConnectionManager)
	cm.net = n
	cm.exists = make(map[string]bool)
	cm.byNodeID = make(map[string]map[uint64]*Connection)
	cm.manage = make(chan *Connection, 64)
	cm.stop = make(chan bool, 1)
	return cm
}

func (cm *ConnectionManager) Start() {
	go cm.manageLoop()
}

func (cm *ConnectionManager) Stop() {
	cm.stop <- true
}

func (cm *ConnectionManager) manageLoop() {
	for {
		select {
		case <-cm.stop:
			for _, c := range cm.conns {
				c.Stop()
			}
			return
		case c := <-cm.manage:
			if cm.exists[c.conn.RemoteAddr().String()] {
				cm.remove(c)
			} else {
				cm.add(c)
			}
		}
	}
}

func (cm *ConnectionManager) remove(con *Connection) {
	for i, f := range cm.conns {
		if f.Socket == con.Socket {
			last := len(cm.conns) - 1
			cm.conns[i] = cm.conns[last]
			cm.conns[last] = nil
			cm.conns = cm.conns[:last]

			if _, ok := cm.byNodeID[con.Address]; ok {
				delete(cm.byNodeID[con.Address], con.NodeID)
			}
			break
		}
	}
}

func (cm *ConnectionManager) add(con *Connection) {
	cm.conns = append(cm.conns, con)
	cm.replaceInto(con)
}

func (cm *ConnectionManager) replaceInto(con *Connection) {
	if con.NodeID == 0 {
		return
	}
	if _, ok := cm.byNodeID[con.Address]; !ok {
		cm.byNodeID[con.Address] = make(map[uint64]*Connection)
	}

	if existing, ok := cm.byNodeID[con.Address][con.NodeID]; ok {
		// TODO add log
		existing.Stop()
	}
	cm.byNodeID[con.Address][con.NodeID] = con
}
