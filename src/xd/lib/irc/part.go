package irc

import "fmt"

type partMsg struct {
	Target IRCTarget
	Reason string
	Source string
}

func (m partMsg) String() string {
	return fmt.Sprintf(":%s PART %s :%s", m.Source, m.Target, m.Reason)
}

func clientPart(line string) (m *partMsg) {
	// TODO: implement
	return
}
