package config

import (
	"github.com/majestrate/XD/lib/configparser"
	"github.com/majestrate/XD/lib/gnutella"
)

type G2Config struct {
	enabled bool
}

// DefaultEnableGnutella says if should we enable gnutella by default
const DefaultEnableGnutella = false

func (c *G2Config) Load(s *configparser.Section) error {
	c.enabled = DefaultEnableGnutella
	if s != nil {
		c.enabled = s.ValueOf("enabled") == "1"
	}
	return nil
}

func (c *G2Config) Save(s *configparser.Section) error {
	if s != nil {
		val := "0"
		if c.enabled {
			val = "1"
		}
		s.Add("enabled", val)
	}
	return nil
}

func (c *G2Config) LoadEnv() {

}

func (c *G2Config) CreateSwarm() *gnutella.Swarm {
	if c.enabled {
		return gnutella.NewSwarm()
	}
	return nil
}
