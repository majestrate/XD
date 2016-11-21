package config

import (
	"xd/lib/configparser"
	"xd/lib/i2p"
)

type I2PConfig struct {
	Addr string
	I2CPOptions map[string]string
}

func (cfg I2PConfig) FromSection(section *configparser.Section) {
	cfg.I2CPOptions = make(map[string]string)
	if section == nil {
		cfg.Addr = i2p.DEFAULT_ADDRESS
	} else {
		cfg.Addr = section.Get("address", i2p.DEFAULT_ADDRESS)
		opts := section.Options()
		for k, v := range opts {
			cfg.I2CPOptions[k] = v
		}
	}
}
