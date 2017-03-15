package config

import (
	"xd/lib/configparser"
)

type Config struct {
	I2P     I2PConfig
	Storage StorageConfig
	RPC     RPCConfig
	Log     LogConfig
}

type configLoadable interface {
	FromSection(s *configparser.Section)
}

// load from file by filename
func (cfg *Config) Load(fname string) (err error) {
	sects := map[string]configLoadable{
		"i2p":     &cfg.I2P,
		"storage": &cfg.Storage,
		"rpc":     &cfg.RPC,
		"log":     &cfg.Log,
	}
	var c *configparser.Configuration
	c, err = configparser.Read(fname)
	for sect, conf := range sects {
		if c == nil {
			conf.FromSection(nil)
		} else {
			s, _ := c.Section(sect)
			conf.FromSection(s)
		}
	}
	return
}
