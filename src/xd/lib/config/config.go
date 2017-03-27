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

// Configurable interface for entity serializable to/from config parser section
type Configurable interface {
	FromSection(s *configparser.Section)
	Options() map[string]string
}

// Load loads a config from file by filename
func (cfg *Config) Load(fname string) (err error) {
	sects := map[string]Configurable{
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

// Save saves a loaded config to file by filename
func (cfg *Config) Save(fname string) (err error) {
	sects := map[string]Configurable{
		"i2p":     &cfg.I2P,
		"storage": &cfg.Storage,
		"rpc":     &cfg.RPC,
		"log":     &cfg.Log,
	}
	c := configparser.NewConfiguration()
	for sect, conf := range sects {
		opts := conf.Options()
		s := c.NewSection(sect)
		for k, v := range opts {
			s.Add(k, v)
		}
	}
	err = configparser.Save(c, fname)
	return
}
