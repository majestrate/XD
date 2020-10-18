package config

import (
	"os"
	"github.com/majestrate/XD/lib/configparser"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/network/i2p"
	"github.com/majestrate/XD/lib/util"
)

type I2PConfig struct {
	Addr            string
	Keyfile         string
	Name            string
	nameWasProvided bool
	I2CPOptions     map[string]string
	Disabled        bool
}

func (cfg *I2PConfig) Load(section *configparser.Section) error {
	cfg.I2CPOptions = make(map[string]string)
	if section == nil {
		cfg.Addr = i2p.DEFAULT_ADDRESS
		cfg.Keyfile = ""
		cfg.Name = util.RandStr(5)
		cfg.Disabled = DisableI2PByDefault
	} else {
		cfg.Disabled = section.Get("disabled", "") == "1"
		cfg.Addr = section.Get("address", i2p.DEFAULT_ADDRESS)
		cfg.Keyfile = section.Get("keyfile", "")
		gen := util.RandStr(5)
		cfg.Name = section.Get("session", gen)
		cfg.nameWasProvided = cfg.Name != gen
		opts := section.Options()
		for k, v := range opts {
			if k == "address" || k == "keyfile" || k == "session" || k == "disabled" {
				continue
			}
			cfg.I2CPOptions[k] = v
		}
	}
	return nil
}

func (cfg *I2PConfig) Save(s *configparser.Section) error {
	opts := make(map[string]string)
	if cfg.I2CPOptions != nil {
		for k, v := range cfg.I2CPOptions {
			opts[k] = v
		}
	}
	opts["address"] = cfg.Addr
	if cfg.Keyfile != "" {
		opts["keyfile"] = cfg.Keyfile
	}
	if cfg.nameWasProvided {
		opts["session"] = cfg.Name
	}
	if cfg.Disabled {
		opts["disabled"] = "1"
	} else {
		opts["disabled"] = "0"
	}
	for k := range opts {
		s.Add(k, opts[k])
	}
	return nil
}

// create an i2p session from this config
func (cfg *I2PConfig) CreateSession() i2p.Session {
	log.Infof("create new i2p session with %s", cfg.Addr)
	return i2p.NewSession(util.RandStr(5), cfg.Addr, cfg.Keyfile, cfg.I2CPOptions)
}

// EnvI2PAddress is the name of the environmental variable to set the i2p address for XD
const EnvI2PAddress = "XD_I2P_ADDRESS"

func (cfg *I2PConfig) LoadEnv() {
	addr := os.Getenv(EnvI2PAddress)
	if addr != "" {
		cfg.Addr = addr
	}
}
