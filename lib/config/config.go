package config

import (
	"github.com/majestrate/XD/lib/configparser"
)

type Config struct {
	LokiNet    LokiNetConfig
	I2P        I2PConfig
	Storage    StorageConfig
	RPC        RPCConfig
	Log        LogConfig
	Bittorrent BittorrentConfig
	Gnutella   G2Config
}

// Configurable interface for entity serializable to/from config parser section
type Configurable interface {
	Load(s *configparser.Section) error
	Save(c *configparser.Section) error
	LoadEnv()
}

// Load loads a config from file by filename
func (cfg *Config) Load(fname string) (err error) {
	sects := map[string]Configurable{
		"lokinet":    &cfg.LokiNet,
		"i2p":        &cfg.I2P,
		"storage":    &cfg.Storage,
		"rpc":        &cfg.RPC,
		"log":        &cfg.Log,
		"bittorrent": &cfg.Bittorrent,
		"gnutella":   &cfg.Gnutella,
	}
	var c *configparser.Configuration
	c, err = configparser.Read(fname)
	for sect, conf := range sects {
		if c == nil {
			err = conf.Load(nil)
		} else {
			s, _ := c.Section(sect)
			err = conf.Load(s)
		}
		conf.LoadEnv()
		if err != nil {
			return
		}
	}
	return
}

// Save saves a loaded config to file by filename
func (cfg *Config) Save(fname string) (err error) {
	sects := map[string]Configurable{
		"lokinet":    &cfg.LokiNet,
		"i2p":        &cfg.I2P,
		"storage":    &cfg.Storage,
		"rpc":        &cfg.RPC,
		"log":        &cfg.Log,
		"bittorrent": &cfg.Bittorrent,
		"gnutella":   &cfg.Gnutella,
	}
	c := configparser.NewConfiguration()
	for sect, conf := range sects {
		s := c.NewSection(sect)
		err = conf.Save(s)
		if err != nil {
			return
		}
	}
	err = configparser.Save(c, fname)
	return
}
