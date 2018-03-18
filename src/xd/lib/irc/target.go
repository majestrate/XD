package irc

type IRCTarget string

func (t IRCTarget) String() string {
	return string(t)
}

func (t IRCTarget) IsChan() bool {
	return len(t) > 0 && t[0] == '#' || t[0] == '$' || t[0] == '#'
}
