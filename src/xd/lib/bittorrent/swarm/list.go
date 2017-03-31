package swarm

import "strings"

type InfohashList []string

func (l InfohashList) Len() int {
	return len(l)
}

func (l InfohashList) Less(i, j int) bool {
	return strings.Compare(l[i], l[j]) < 0
}

func (l *InfohashList) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

type TorrentsList struct {
	Infohashes InfohashList
}
