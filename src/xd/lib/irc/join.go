package irc

import "fmt"

type joinMsg struct {
	Target IRCTarget
	Source string
}

func (j joinMsg) String() string {
	return fmt.Sprintf(":%s JOIN :%s", j.Source, j.Target.String())
}

func clientJoin(line string) (m *joinMsg) {
	// TODO: implement
	return
}
