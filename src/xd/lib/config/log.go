package config

import (
	"xd/lib/configparser"
)

type LogConfig struct {
	Level string
}

func (cfg *LogConfig) FromSection(s *configparser.Section) {

	cfg.Level = "info"
	if s != nil {
		cfg.Level = s.Get("level", "info")
	}
}

func (cfg *LogConfig) Options() map[string]string {
	return map[string]string{
		"level": cfg.Level,
	}
}
