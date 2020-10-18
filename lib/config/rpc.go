package config

import (
	"os"
	"github.com/majestrate/XD/lib/configparser"
)

type RPCConfig struct {
	Enabled      bool
	Bind         string
	ExpectedHost string
	Auth         bool
	Username     string
	Password     string
}

const DefaultRPCAddr = "127.0.0.1:1776"
const DefaultRPCHost = "127.0.0.1"
const DefaultRPCAuth = "0"

func (cfg *RPCConfig) Load(s *configparser.Section) error {
	if s != nil {
		cfg.ExpectedHost = s.Get("host", DefaultRPCHost)
		cfg.Bind = s.Get("bind", DefaultRPCAddr)
		cfg.Enabled = s.Get("enabled", "1") == "1"
		cfg.Auth = s.Get("auth", DefaultRPCAuth) == "1"
		cfg.Username = s.Get("username", "")
		cfg.Password = s.Get("password", "")
	}
	if cfg.Bind == "" {
		cfg.Bind = DefaultRPCAddr
		cfg.Enabled = true
	}
	if cfg.ExpectedHost == "" {
		cfg.ExpectedHost = DefaultRPCHost
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
	if cfg.ExpectedHost != "" {
		opts["host"] = cfg.ExpectedHost
	}

	if cfg.Auth && cfg.Username != "" && cfg.Password != "" {
		opts["auth"] = "1"
		opts["username"] = cfg.Username
		opts["password"] = cfg.Password
	}

	for k := range opts {
		s.Add(k, opts[k])
	}

	return nil
}

const EnvRPCAddr = "XD_RPC_ADDRESS"
const EnvRPCHost = "XD_RPC_HOST"

func (cfg *RPCConfig) LoadEnv() {
	addr := os.Getenv(EnvRPCAddr)
	if addr != "" {
		cfg.Bind = addr
	}
	host := os.Getenv(EnvRPCHost)
	if host != "" {
		cfg.ExpectedHost = host
	}
}
