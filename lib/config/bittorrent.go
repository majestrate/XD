package config

import (
	"fmt"
	"os"
	"strconv"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
	"github.com/majestrate/XD/lib/configparser"
	"github.com/majestrate/XD/lib/gnutella"
	"github.com/majestrate/XD/lib/storage"
	"github.com/majestrate/XD/lib/util"
)

const DefaultTorrentQueueSize = 0
const DefaultOpentrackerFilename = "trackers.ini"

type TrackerConfig struct {
	Trackers map[string]string
	FileName string
}

func (c *TrackerConfig) Save() (err error) {
	if c.Trackers == nil || len(c.Trackers) == 0 {
		c.Trackers = DefaultOpenTrackers
	}
	cfg := configparser.NewConfiguration()
	for sect := range c.Trackers {
		s := cfg.NewSection(sect)
		s.Add("url", c.Trackers[sect])
	}
	err = configparser.Save(cfg, c.FileName)
	return
}

func (c *TrackerConfig) Load() (err error) {

	if len(c.FileName) == 0 {
		c.FileName = DefaultOpentrackerFilename
	}

	// create defaults
	if !util.CheckFile(c.FileName) {
		err = c.Save()
	}

	if err == nil {
		var cfg *configparser.Configuration
		cfg, err = configparser.Read(c.FileName)
		if err == nil {
			var sects []*configparser.Section
			sects, err = cfg.AllSections()
			if err == nil {
				if c.Trackers == nil {
					c.Trackers = make(map[string]string)
				}
				for idx := range sects {
					if sects[idx].Exists("url") {
						c.Trackers[sects[idx].Name()] = sects[idx].ValueOf("url")
					}
				}
			}
		}
	}
	return
}

type BittorrentConfig struct {
	DHT              bool
	PEX              bool
	OpenTrackers     TrackerConfig
	PieceWindowSize  int
	Swarms           int
	TorrentQueueSize int
}

func (c *BittorrentConfig) Load(s *configparser.Section) error {
	c.OpenTrackers.FileName = DefaultOpentrackerFilename
	c.PieceWindowSize = swarm.DefaultMaxParallelRequests
	c.TorrentQueueSize = DefaultTorrentQueueSize
	c.PEX = true
	c.Swarms = 1
	if s != nil {
		c.DHT = s.Get("dht", "0") == "1"
		c.PEX = s.Get("pex", "1") == "1"
		c.OpenTrackers.FileName = s.Get("tracker-config", c.OpenTrackers.FileName)
		var e error
		c.PieceWindowSize, e = strconv.Atoi(s.Get("piece-window", fmt.Sprintf("%d", swarm.DefaultMaxParallelRequests)))
		if e != nil {
			c.PieceWindowSize = swarm.DefaultMaxParallelRequests
		}
		c.Swarms, e = strconv.Atoi(s.Get("swarms", "1"))
		if e != nil {
			return e
		}
		c.TorrentQueueSize, e = strconv.Atoi(s.Get("max-torrents", "0"))
		if e != nil {
			return e
		}
	}
	return c.OpenTrackers.Load()
}

func (c *BittorrentConfig) Save(s *configparser.Section) error {
	if c.PEX {
		s.Add("pex", "1")
	} else {
		s.Add("pex", "0")
	}

	if c.DHT {
		s.Add("dht", "1")
	} else {
		s.Add("dht", "0")
	}

	s.Add("swarms", fmt.Sprintf("%d", c.Swarms))

	s.Add("tracker-config", c.OpenTrackers.FileName)

	s.Add("max-torrents", fmt.Sprintf("%d", c.TorrentQueueSize))

	return c.OpenTrackers.Save()
}

const EnvOpenTracker = "XD_OPENTRACKER_URL"

func (cfg *BittorrentConfig) LoadEnv() {
	url := os.Getenv(EnvOpenTracker)
	if url != "" {
		cfg.OpenTrackers.Trackers = map[string]string{
			"default": url,
		}
	}
}

func (c *BittorrentConfig) CreateSwarm(st storage.Storage, gnutella *gnutella.Swarm) *swarm.Swarm {
	sw := swarm.NewSwarm(st, gnutella)
	for name := range c.OpenTrackers.Trackers {
		sw.AddOpenTracker(c.OpenTrackers.Trackers[name])
	}
	sw.Torrents.MaxReq = c.PieceWindowSize
	sw.Torrents.QueueSize = c.TorrentQueueSize
	return sw
}
