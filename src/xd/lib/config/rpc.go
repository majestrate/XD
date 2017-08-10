package config

import (
	"os"
	"xd/lib/configparser"
)

type RPCConfig struct {
	Enabled bool
	Bind    string
	// TODO: authentication
}

const DefaultRPCAddr = "127.0.0.1:1488"

func (cfg *RPCConfig) Load(s *configparser.Section) error {
	if s != nil {
		cfg.Bind = s.Get("bind", DefaultRPCAddr)
		cfg.Enabled = s.Get("enabled", "1") == "1"
	}
	if cfg.Bind == "" {
		cfg.Bind = DefaultRPCAddr
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

const EnvRPCAddr = "XD_RPC_ADDRESS"

func (cfg *RPCConfig) LoadEnv() {
	addr := os.Getenv(EnvRPCAddr)
	if addr != "" {
		cfg.Bind = addr
	}
}
