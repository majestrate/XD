package config

import (
	"github.com/majestrate/XD/lib/configparser"
	"os"
)

const EnvLogLevel = "XD_LOG_LEVEL"
const EnvLogPProf = "XD_PPROF"

type LogConfig struct {
	Level string
	Pprof bool
}

func (cfg *LogConfig) Load(s *configparser.Section) error {

	cfg.Level = "info"
	if s != nil {
		cfg.Level = s.Get("level", "info")
		cfg.Pprof = s.Get("pprof", "0") == "1"
	}

	return nil
}

func (cfg *LogConfig) Save(s *configparser.Section) error {
	lvl := "0"
	if cfg.Pprof {
		lvl = "1"
	}
	s.Add("level", cfg.Level)
	s.Add("pprof", lvl)
	return nil
}

func (cfg *LogConfig) LoadEnv() {
	lvl := os.Getenv(EnvLogLevel)
	if lvl != "" {
		cfg.Level = lvl
	}
	lvl = os.Getenv(EnvLogPProf)
	if lvl != "" {
		cfg.Pprof = lvl == "1"
	}
}
