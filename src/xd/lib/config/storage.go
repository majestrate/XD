package config

import (
	"os"
	"path/filepath"
	"xd/lib/configparser"
	"xd/lib/fs"
	"xd/lib/storage"
)

// EnvRootDir is the name of the environmental variable to set the root storage directory at runtime
const EnvRootDir = "XD_HOME"

type SFTPConfig struct {
	Enabled      bool
	Username     string
	Hostname     string
	Keyfile      string
	RemotePubkey string
	Port         int
}

func (cfg *SFTPConfig) Load(s *configparser.Section) error {
	cfg.Username = s.Get("sftp_user", "")
	cfg.Hostname = s.Get("sftp_host", "")
	cfg.Keyfile = s.Get("sftp_keyfile", "")
	cfg.RemotePubkey = s.Get("sftp_remotekey", "")
	cfg.Port = s.GetInt("sftp_port", 22)
	return nil
}

func (cfg *SFTPConfig) Save(s *configparser.Section) error {
	return nil
}

func (cfg *SFTPConfig) LoadEnv() {

}

func (cfg *SFTPConfig) ToFS() fs.Driver {
	return fs.SFTP(cfg.Username, cfg.Hostname, cfg.Keyfile, cfg.RemotePubkey, cfg.Port)
}

type StorageConfig struct {
	// downloads directory
	Downloads string
	// metadata directory
	Meta string
	// root directory
	Root string
	// sftp config
	SFTP SFTPConfig
}

func (cfg *StorageConfig) Load(s *configparser.Section) error {

	if cfg.Root == "" {
		cfg.Root = "storage"
		if s != nil {
			cfg.Root = s.Get("rootdir", cfg.Root)
		}
	}

	cfg.Meta = filepath.Join(cfg.Root, "metadata")

	cfg.Downloads = filepath.Join(cfg.Root, "downloads")

	if s != nil {
		cfg.Downloads = s.Get("downloads", cfg.Downloads)
	}

	if s != nil {
		cfg.SFTP.Enabled = s.Get("sftp", "0") == "1"
	}
	if cfg.SFTP.Enabled {
		return cfg.SFTP.Load(s)
	}
	return nil
}

func (cfg *StorageConfig) Save(s *configparser.Section) error {

	s.Add("rootdir", cfg.Root)
	s.Add("metadata", cfg.Meta)
	s.Add("downloads", cfg.Downloads)
	return nil
}

func (cfg *StorageConfig) LoadEnv() {
	dir := os.Getenv(EnvRootDir)
	if dir != "" {
		cfg.Root = dir
	}
}

func (cfg *StorageConfig) CreateStorage() storage.Storage {

	st := &storage.FsStorage{
		DataDir: cfg.Downloads,
		MetaDir: cfg.Meta,
		FS:      fs.STD,
	}
	if cfg.SFTP.Enabled {
		st.FS = cfg.SFTP.ToFS()
	}
	return st
}
