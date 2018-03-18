package irc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/textproto"
	"strings"
	"xd/lib/crypto"
	"xd/lib/util"
	"xd/lib/version"
)

var ErrNotRegistered = errors.New("not registered")

type Conn struct {
	c        *textproto.Conn
	sendChnl chan string
	sk       *crypto.SecretKey
}

func (c *Conn) Nick() string {
	return hex.EncodeToString(c.sk.ToPublic().Bytes())[0:10]
}

func (c *Conn) blockingHandshake(from string) (err error) {
	var cookieAccepted bool
	pingCookie := util.RandStr(5)
	var nick string
	var user string
	var lines int
	err = c.c.PrintfLine(":%s PING %s", from, pingCookie)
	for err == nil {
		var line string
		lines++
		line, err = c.c.ReadLine()
		if line == "PONG "+pingCookie || line == "PONG :"+pingCookie {
			cookieAccepted = true
		}
		if strings.HasPrefix(line, "NICK ") {
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				nick = parts[1]
			}
		}
		if strings.HasPrefix(line, "USER ") {
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				user = parts[1]
			}
		}
		if user != "" && nick != "" && cookieAccepted {
			break
		}
		if lines > 10 {
			err = ErrNotRegistered
		}
	}
	if err == nil {
		c.c.PrintfLine(":%s 001 Welcome to the Internet Relay Network %s!%s@anonymous", from, nick, user)
		c.c.PrintfLine(":%s 002 Your host is XD, running %s", from, version.Version())
		c.c.PrintfLine(":%s 003 This server was created sometime", from)
		c.c.PrintfLine(":%s 004 XD %s %s %s", from, version.Version(), umodes, cmodes)
		c.c.PrintfLine(":%s %d %s :%s", from, RPL_NOMOTD, nick, "no MOTD")
		// rename user
		err = c.c.PrintfLine(":%s %s NICK %s", from, nick, c.Nick())
	}
	return
}

func (c *Conn) Quit(from, reason string) {
	c.c.PrintfLine(":%s QUIT :%s", from, reason)
	c.c.Close()
}

func (c *Conn) Numeric(from string, num int, trailing string) {
	c.SendLine(fmt.Sprintf(":%s %d %s %s", from, num, c.Nick, trailing))
}

func (c *Conn) runWriter() {
	for {
		err := c.c.PrintfLine("%s", <-c.sendChnl)
		if err != nil {
			return
		}
	}
}

func (c *Conn) SendLine(line string) {
	c.sendChnl <- line
}

func (c *Conn) runChat(from string, recvs2sline func(string, *Conn)) {
	go c.runWriter()
	for {
		line, err := c.c.ReadLine()
		if err != nil {
			return
		}
		ping := clientPing(line)
		if ping != nil {
			c.SendLine(ping.Pong(from))
			continue
		}

		quit := clientQuit(line)
		if quit != nil {
			c.Quit(from, quit.Reason)
			return
		}

		part := clientPart(line)
		if part != nil {
			recvs2sline(part.String(), c)
			continue
		}
		join := clientJoin(line)
		if join != nil {
			join.Source = c.Nick()
			recvs2sline(join.String(), c)
			continue
		}
		pm := clientPrivmsg(line)
		if pm != nil {
			pm.Source = IRCTarget(c.Nick())
			recvs2sline(pm.String(), c)
			continue
		}
	}
}
