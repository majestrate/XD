//go:build lokinet
// +build lokinet

package config

const DisableLokinetByDefault = false
const DisableI2PByDefault = true

var DefaultOpenTrackers = map[string]string{
	"default": "http://probably.loki:6680/announce",
}
