package config

import (
	"fmt"
	"xd/lib/configparser"
	"xd/lib/network"
	"xd/lib/network/tor"
)

type TorConfig struct {
	Addr     string
	Net      string
	Privkey  string
	Password string
	ExtPort  int
	Enabled  bool
}

const DefaultTorAddr = "127.0.0.1:9050"
const DefaultTorNet = "tcp"
const DefaultTorExtPort = 6889

func (cfg *TorConfig) Load(section *configparser.Section) error {
	if section == nil {
		cfg.Addr = DefaultTorAddr
		cfg.Net = DefaultTorNet
		cfg.Privkey = ""
		cfg.Enabled = false
		cfg.ExtPort = DefaultTorExtPort
	} else {
		cfg.Addr = section.Get("addr", DefaultTorAddr)
		cfg.Net = section.Get("net", DefaultTorNet)
		cfg.Privkey = section.Get("privkey", "")
		cfg.Enabled = section.Get("enable", "0") == "1"
		cfg.Password = section.Get("password", "")
		cfg.ExtPort = section.GetInt("extport", DefaultTorExtPort)
	}
	return nil
}

func (cfg *TorConfig) Save(s *configparser.Section) error {
	s.Add("addr", cfg.Addr)
	s.Add("net", cfg.Net)
	s.Add("privkey", cfg.Privkey)
	if cfg.Enabled {
		s.Add("enable", "1")
	}
	s.Add("extport", fmt.Sprintf("%d", cfg.ExtPort))
	return nil
}

func (cfg *TorConfig) LoadEnv() {
}

func (cfg *TorConfig) CreateSession() network.Network {
	return tor.CreateSession(cfg.Net, cfg.Addr, cfg.Privkey, cfg.Password, cfg.ExtPort)
}
