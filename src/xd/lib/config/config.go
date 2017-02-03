package config

import (
	"xd/lib/configparser"
)

type Config struct {
	I2P     I2PConfig
	Storage StorageConfig
	RPC     RPCConfig
}

// load from file by filename
func (cfg *Config) Load(fname string) (err error) {
	var c *configparser.Configuration
	c, err = configparser.Read(fname)
	if err == nil {
		s, _ := c.Section("i2p")
		cfg.I2P.FromSection(s)
		s, _ = c.Section("storage")
		cfg.Storage.FromSection(s)
		s, _ = c.Section("rpc")
		cfg.RPC.FromSection(s)
	} else {
		cfg.I2P.FromSection(nil)
		cfg.Storage.FromSection(nil)
	}
	return
}
