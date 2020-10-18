package gnutella

type Swarm struct {
	activeConns []*Conn
}

func (sw *Swarm) AddInboundPeer(conn *Conn) {
	sw.activeConns = append(sw.activeConns, conn)
}

func (sw *Swarm) Close() error {
	for _, conn := range sw.activeConns {
		conn.Close()
	}
	sw.activeConns = []*Conn{}
	return nil
}

func NewSwarm() *Swarm {
	return &Swarm{}
}
