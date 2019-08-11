// +build lokinet

package config

const DisableLokinetByDefault = false
const DisableI2PByDefault = true

var DefaultOpenTrackers = map[string]string{
	"ICX": "http://icxqqcpd3sfkjbqifn53h7rmusqa1fyxwqyfrrcgkd37xcikwa7y.loki:6680/announce",
}
