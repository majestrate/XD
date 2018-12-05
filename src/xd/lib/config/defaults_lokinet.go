// +build lokinet

package config

const DisableLokinetByDefault = false
const DisableI2PByDefault = true

var DefaultOpenTrackers = map[string]string{
	"7oki": "http://7okic5x5do3uh3usttnqz9ek3uuoemdrwzto1hciwim9f947or6y.loki:6680/announce",
}
