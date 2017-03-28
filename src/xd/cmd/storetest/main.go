package main

import (
	"fmt"
	"os"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/config"
	"xd/lib/log"
	"xd/lib/storage"
)

func check(t storage.Torrent, bf *bittorrent.Bitfield) (err error) {
	name := t.Name()
	i := t.MetaInfo()
	np := i.Info.NumPieces()
	log.Infof("checking %s", name)
	log.Infof("%d pieces, %d bytes per piece, %d bytes total", np, i.Info.PieceLength, i.TotalSize())
	idx := uint32(0)
	skipped := uint32(0)
	for idx < np {
		if bf == nil || bf.Has(idx) {
			l := i.Info.PieceLength
			if idx == np-1 {
				l -= uint32((uint64(np) * uint64(i.Info.PieceLength)) - i.TotalSize())
			}
			var pc *common.PieceData
			r := &common.PieceRequest{
				Index:  idx,
				Length: l,
			}
			pc, err = t.GetPiece(r)
			if err == nil {
				if pc == nil {
					log.Errorf("get piece %d returned nil", idx)
				} else {
					err = t.PutPiece(pc)
				}
			}
			if err == nil {
				var pcAfter *common.PieceData
				pcAfter, err = t.GetPiece(r)
				if err == nil {
					if !pc.Equals(pcAfter) {
						log.Errorf("piece %d storage missmatch", idx)
						return
					}
				} else {
					log.Errorf("get piece %d returned nil after store", idx)
					return
				}
			} else {
				log.Errorf("failed to put piece %d for %s: %s", idx, name, err)
				return
			}
		} else {
			skipped++
		}
		idx++
	}
	log.Infof("done checking %s, skipped %d of %d pieces", name, skipped, np)
	return
}

func main() {
	conf := new(config.Config)
	fname := "torrents.ini"
	if len(os.Args) > 1 {
		fname = os.Args[1]
	}
	if fname == "-h" || fname == "--help" {
		fmt.Fprintf(os.Stdout, "usage: %s [config.ini]\n", os.Args[0])
		return
	}
	err := conf.Load(fname)
	if err != nil {
		log.Errorf("failed to load config: %s", err)
		return
	}
	log.SetLevel(conf.Log.Level)
	st := conf.Storage.CreateStorage()
	var ts []storage.Torrent
	for _, t := range st.PollNewTorrents() {
		err = check(t, nil)
		if err != nil {
			return
		}
	}

	ts, err = st.OpenAllTorrents()
	if err != nil {
		log.Errorf("failed to open torrents: %s", err)
		return
	}

	for _, t := range ts {
		err = check(t, t.Bitfield())
		if err != nil {
			return
		}
	}
}
