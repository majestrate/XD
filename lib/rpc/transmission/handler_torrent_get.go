package transmission

import (
	"fmt"
	"github.com/majestrate/XD/lib/bittorrent/swarm"
)

func TorrentGet(sw *swarm.Swarm, args Args) (resp Response) {
	resp.Args = make(Args)
	i_fields, ok := args["fields"]
	var err error
	if ok {
		ids := getTorrentIDs(sw.Torrents.TorrentIDs, args)
		var torrents []tgResp

		for _, id := range ids {
			r := make(tgResp)
			f_slice, ok := i_fields.([]interface{})
			if !ok {
				resp.Result = "fields is not an array"
				return
			}
			t := sw.Torrents.GetTorrentByID(int64(id))
			if t != nil {
				for _, f := range f_slice {
					field, ok := f.(string)
					if ok {
						h, ok := tgFieldHandlers[field]
						if ok {
							err = h(field, t, &r)
							if err != nil {
								break
							}
						} else {
							resp.Result = fmt.Sprintf("field '%s' not implemented", field)
							return
						}
					} else {
						resp.Result = "field is not a string"
						return
					}
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
