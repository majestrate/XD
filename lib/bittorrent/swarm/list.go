package swarm

import "github.com/majestrate/XD/lib/util"

type InfohashList []string

func (l InfohashList) Len() int {
	return len(l)
}

func (l InfohashList) Less(i, j int) bool {
	return util.StringCompare(l[i], l[j]) < 0
}

func (l *InfohashList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

type TorrentsList struct {
	Infohashes InfohashList
}
