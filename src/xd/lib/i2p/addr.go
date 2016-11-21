package i2p

// implements net.Addr
type I2PAddr string

func (a I2PAddr) Network() string {
	return "i2p"
}

func (a I2PAddr) String() string {
	return string(a)
}

