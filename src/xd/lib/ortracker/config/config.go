package config

import (
	"xd/lib/config"
	"xd/lib/configparser"
	"xd/lib/log"
)

type Config struct {
	Tor config.TorConfig
	I2P config.I2PConfig
}

func (cfg *Config) Load(fname string) (err error) {
	sects := map[string]config.Configurable{
		"i2p": &cfg.I2P,
		"tor": &cfg.Tor,
	}
	var c *configparser.Configuration
	c, err = configparser.Read(fname)
	for sect, conf := range sects {
		conf.LoadEnv()
		if c == nil {
			err = conf.Load(nil)
		} else {
			log.Debugf("found section %s", sect)
			s, _ := c.Section(sect)
			err = conf.Load(s)
		}
		if err != nil {
			return
		}
	}
	return
}

// Save saves a loaded config to file by filename
func (cfg *Config) Save(fname string) (err error) {
	sects := map[string]config.Configurable{
		"i2p": &cfg.I2P,
		"tor": &cfg.Tor,
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
