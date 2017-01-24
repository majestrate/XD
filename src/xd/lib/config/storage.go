package config

import (
	"path/filepath"
	"xd/lib/configparser"
	"xd/lib/storage"
	"xd/lib/util"
)

type StorageConfig struct {
	// downloads directory
	Downloads string
	// metadata directory
	Meta string
	// root directory
	Root string
}

func (cfg *StorageConfig) FromSection(s *configparser.Section) {

	cfg.Root = "XD"
	if s != nil {
		cfg.Root = s.Get("rootdir", cfg.Root)
	}

	cfg.Meta = filepath.Join(cfg.Root, "metadata")

	cfg.Downloads = filepath.Join(cfg.Root, "downloads")
	if s != nil {
		cfg.Downloads = s.Get("downloads", cfg.Downloads)
	}
}

func (cfg *StorageConfig) CreateStorage() storage.Storage {
	util.EnsureDir(cfg.Root)
	util.EnsureDir(cfg.Downloads)
	util.EnsureDir(cfg.Meta)
	return &storage.FsStorage{
		DataDir: cfg.Downloads,
		MetaDir: cfg.Meta,
	}
}
