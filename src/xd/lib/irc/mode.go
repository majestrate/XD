package irc

import (
	"strings"
)

type modeMSG struct {
	mode string
}

func (m modeMSG) String() string {
	return ""
}

func clientMode(line string) (m *modeMSG) {
	if strings.HasPrefix(strings.ToUpper(line), "MODE ") && len(line) > 5 {
		m = &modeMSG{
			mode: line[5:],
		}
	}
	return
}
