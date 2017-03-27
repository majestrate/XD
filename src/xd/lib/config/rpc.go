package config

import (
	"xd/lib/configparser"
)

type RPCConfig struct {
	Enabled bool
	Bind    string
	// TODO: authentication
}

func (cfg *RPCConfig) FromSection(s *configparser.Section) {
	if s == nil {
		return
	}
	cfg.Bind = s.Get("bind", "127.0.0.1:1188")
	cfg.Enabled = s.Get("enabled", "1") == "1"
}

func (cfg *RPCConfig) Options() map[string]string {
	enabled := "0"
	if cfg.Enabled {
		enabled = "1"
	}
	return map[string]string{
		"bind":    cfg.Bind,
		"enabled": enabled,
	}
}
