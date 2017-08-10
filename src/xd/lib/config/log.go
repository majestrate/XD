package config

import (
	"os"
	"xd/lib/configparser"
)

const EnvLogLevel = "XD_LOG_LEVEL"

type LogConfig struct {
	Level string
}

func (cfg *LogConfig) Load(s *configparser.Section) error {

	cfg.Level = "info"
	if s != nil {
		cfg.Level = s.Get("level", "info")
	}
	return nil
}

func (cfg *LogConfig) Save(s *configparser.Section) error {
	s.Add("level", cfg.Level)
	return nil
}

func (cfg *LogConfig) LoadEnv() {
	lvl := os.Getenv(EnvLogLevel)
	if lvl != "" {
		cfg.Level = lvl
	}
}
