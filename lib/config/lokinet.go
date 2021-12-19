package config

import (
	"github.com/majestrate/XD/lib/configparser"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/network/inet"
	"os"
)

type LokiNetConfig struct {
	DNSAddr  string
	Port     string
	Disabled bool
}

func (cfg *LokiNetConfig) Load(section *configparser.Section) error {
	if section == nil {
		cfg.DNSAddr = inet.DefaultDNSAddr
		cfg.Port = inet.DefaultPort
		cfg.Disabled = DisableLokinetByDefault
	} else {
		cfg.Disabled = section.Get("disabled", "") == "1"
		cfg.DNSAddr = section.Get("dns", inet.DefaultDNSAddr)
		cfg.Port = section.Get("port", inet.DefaultPort)
	}
	return nil
}

func (cfg *LokiNetConfig) Save(s *configparser.Section) error {
	opts := make(map[string]string)
	opts["dns"] = cfg.DNSAddr
	if cfg.Disabled {
		opts["disabled"] = "1"
	}
	for k := range opts {
		s.Add(k, opts[k])
	}
	return nil
}

// create a network session from this config
func (cfg *LokiNetConfig) CreateSession() (*inet.Session, error) {
	log.Infof("create new session on lokinet")
	return inet.NewSession(cfg.Port, cfg.DNSAddr)
}

func (cfg *LokiNetConfig) LoadEnv() {
	addr := os.Getenv("LOKINET_DNS")
	if addr != "" {
		cfg.DNSAddr = addr
	}
}
