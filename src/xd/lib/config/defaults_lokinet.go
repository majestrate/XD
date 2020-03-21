// +build lokinet

package config

const DisableLokinetByDefault = false
const DisableI2PByDefault = true

var DefaultOpenTrackers = map[string]string{
	"default": "http://azxoy94ffnnqa54pqxhmoc6geaigdrqqkwxzk3jiafazo8x8a73o.loki:6680/announce",
}
