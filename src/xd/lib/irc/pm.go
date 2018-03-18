package irc

import (
	"fmt"
	"strings"
)

type privMSG struct {
	Source  IRCTarget
	Target  IRCTarget
	Message string
}

func (p *privMSG) String() string {
	return fmt.Sprintf(":%s PRIVMSG %s :%s", p.Source, p.Target, p.Message)
}

func clientPrivmsg(line string) (p *privMSG) {
	if strings.HasPrefix(strings.ToUpper(line), "PRIVMSG ") {
		parts := strings.Split(line, " ")
		if len(parts) > 3 {
			p = &privMSG{
				Target: IRCTarget(parts[1]),
			}
			idx := strings.Index(line, p.Target.String())
			p.Message = line[idx+len(p.Target):]
			if len(p.Message) > 0 && p.Message[0] == ' ' {
				p.Message = p.Message[1:]
				if len(p.Message) > 0 && p.Message[0] == ':' {
					p.Message = p.Message[1:]
				}
			} else {
				p = nil
			}
		}
	}
	return
}

func getPrivmsg(line string) (p *privMSG) {
	return
}
