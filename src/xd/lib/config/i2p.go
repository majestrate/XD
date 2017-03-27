package config

import (
	"xd/lib/configparser"
	"xd/lib/log"
	"xd/lib/network/i2p"
	"xd/lib/util"
)

type I2PConfig struct {
	Addr             string
	Keyfile          string
	Name             string
	nameWasGenerated bool
	I2CPOptions      map[string]string
}

func (cfg *I2PConfig) FromSection(section *configparser.Section) {
	cfg.I2CPOptions = make(map[string]string)
	if section == nil {
		cfg.Addr = i2p.DEFAULT_ADDRESS
		cfg.Keyfile = ""
		cfg.Name = util.RandStr(5)
	} else {
		cfg.Addr = section.Get("address", i2p.DEFAULT_ADDRESS)
		cfg.Keyfile = section.Get("keyfile", "")
		gen := util.RandStr(5)
		cfg.Name = section.Get("session", gen)
		cfg.nameWasGenerated = cfg.Name == gen
		opts := section.Options()
		for k, v := range opts {
			if k == "address" || k == "keyfile" || k == "session" {
				continue
			}
			cfg.I2CPOptions[k] = v
		}
	}
}

func (cfg *I2PConfig) Options() map[string]string {
	opts := make(map[string]string)
	if cfg.I2CPOptions != nil {
		for k, v := range cfg.I2CPOptions {
			opts[k] = v
		}
	}
	opts["address"] = cfg.Addr
	opts["keyfile"] = cfg.Keyfile
	if !cfg.nameWasGenerated {
		opts["session"] = cfg.Name
	}
	return opts
}

// create an i2p session from this config
func (cfg *I2PConfig) CreateSession() i2p.Session {
	log.Infof("create new i2p session with %s", cfg.Addr)
	return i2p.NewSession(cfg.Name, cfg.Addr, cfg.Keyfile)
}
