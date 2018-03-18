package irc

import (
	"fmt"
	"strings"
)

type pingMSG struct {
	param string
}

func clientPing(line string) (p *pingMSG) {
	if strings.HasPrefix(strings.ToUpper(line), "PING ") && len(line) > 5 {
		param := line[5:]
		if param[0] == ':' && len(param) > 1 {
			param = param[1:]
		}
		p = &pingMSG{
			param: param,
		}
	}
	return
}

func (msg *pingMSG) Pong(from string) string {
	return fmt.Sprintf(":%s PONG :%s", from, msg.param)
}
