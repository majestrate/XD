package transmission

import (
	"fmt"
	"xd/lib/bittorrent/swarm"
)

func TorrentGet(sw *swarm.Swarm, args Args) (resp Response) {
	resp.Args = make(Args)
	i, ok := args["fields"]
	var err error
	if ok {
		var ids TorrentIDArray
		ids_i, ok := args["ids"]
		if ok {
			for _, id := range ids_i.([]interface{}) {
				tid, ok := id.(int64)
				if ok {
					ids = append(ids, TorrentID(tid))
				}
			}
		} else {
			tids := sw.Torrents.TorrentIDs()
			for tid := range tids {
				ids = append(ids, TorrentID(tid))
			}
		}
		var torrents []tgResp

		for _, id := range ids {
			r := make(tgResp)
			for _, f := range i.([]interface{}) {

				field, ok := f.(string)
				if ok {
					h, ok := tgFieldHandlers[field]
					if ok {
						err = h(sw, &r, id)
						if err != nil {
							break
						}
					} else {
						resp.Result = fmt.Sprintf("field '%s' not implemented", field)
						return
					}
				} else {
					resp.Result = fmt.Sprintf("field is not a string")
					return
				}
			}
			if err == nil {
				torrents = append(torrents, r)
			} else {
				resp.Result = err.Error()
				return
			}
		}
		resp.Args["torrents"] = torrents
		resp.Result = Success
	} else {
		resp.Result = "no fields provided"
	}
	return
}
