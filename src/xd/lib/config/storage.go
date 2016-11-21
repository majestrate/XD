package config

import (
	"path/filepath"
	"xd/lib/configparser"
)

type StorageConfig struct {
	// directory for leeching files
	LeechPath string
	// directory for seeding files
	SeedPath string
}


func (cfg StorageConfig) FromSection(s *configparser.Section) {
	cfg.LeechPath = filepath.Join("torrents", "downloads")
	cfg.SeedPath = filepath.Join("torrents", "seeding")
	if s != nil {
		cfg.LeechPath = s.Get("download_dir", cfg.LeechPath)
		cfg.SeedPath = s.Get("seed_dir", cfg.SeedPath)
	}
}
