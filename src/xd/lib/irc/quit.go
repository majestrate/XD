package irc

import (
	"strings"
)

type quitMSG struct {
	Reason string
}

func clientQuit(line string) (m *quitMSG) {
	if strings.HasPrefix(strings.ToUpper(line), "QUIT ") {
		m = &quitMSG{
			Reason: line[5:],
		}
		if len(m.Reason) > 0 && m.Reason[0] == ':' {
			m.Reason = m.Reason[1:]
		}
		if m.Reason == "" {
			m.Reason = "Quitting"
		}
	}
	if strings.ToUpper(line) == "QUIT" {
		m = &quitMSG{
			Reason: "Quitting",
		}
	}
	return
}
