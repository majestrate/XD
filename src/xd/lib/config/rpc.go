package config

import (
	"xd/lib/configparser"
)

type RPCConfig struct {
	Enabled bool
	Bind    string
	// TODO: authentication
}

func (cfg *RPCConfig) Load(s *configparser.Section) error {
	if s != nil {
		cfg.Bind = s.Get("bind", "127.0.0.1:1188")
		cfg.Enabled = s.Get("enabled", "1") == "1"
	}
	return nil
}

func (cfg *RPCConfig) Save(s *configparser.Section) error {
	enabled := "1"
	if !cfg.Enabled {
		enabled = "0"
	}
	opts := map[string]string{
		"enabled": enabled,
	}
	if cfg.Bind != "" {
		opts["bind"] = cfg.Bind
	}

	for k := range opts {
		s.Add(k, opts[k])
	}

	return nil
}
