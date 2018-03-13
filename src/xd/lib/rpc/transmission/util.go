package transmission

func getTorrentIDs(getActiveIDs func() map[int64]string, args Args) (ids TorrentIDArray) {
	ids_i, ok := args["ids"]
	if ok {
		ids_slice, ok := ids_i.([]interface{})
		if ok {
			for _, id := range ids_slice {
				tid, ok := id.(int64)
				if ok {
					ids = append(ids, TorrentID(tid))
				}
			}
		} else {
			ids_str, ok := ids_i.(string)
			if ok {
				if ids_str == idRecentlyActive {
					tids := getActiveIDs()
					for tid := range tids {
						ids = append(ids, TorrentID(tid))
					}
				}
			} else {
				ids_int, ok := ids_i.(int64)
				if ok {
					ids = append(ids, TorrentID(ids_int))
				}
			}
		}
	}
	return
}
