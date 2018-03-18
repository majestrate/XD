package irc

import (
	"net"
	"net/textproto"
	"xd/lib/crypto"
	"xd/lib/log"
	"xd/lib/sync"
)

type RemoteHandler interface {
	SendPrivMsg(name, msg string)
	SendBCMsg(target, msg string)
	HasUser(name string) bool
}

type Server struct {
	localUsers    sync.Map
	Name          string
	NickLen       int
	ChanLen       int
	ErrorChannel  string
	RemoteHandler RemoteHandler
	lineChnl      chan lineEvent
	quit          bool
}

func (s *Server) visit(name string, v func(*Conn)) {
	c, ok := s.localUsers.Load(name)
	if ok {
		conn := c.(*Conn)
		v(conn)
	}
}

func (s *Server) hasNick(name string) (has bool) {
	s.visit(name, func(_ *Conn) {
		has = true
	})
	return
}

func (s *Server) sendToNick(name string) {
}

func (s *Server) broadcastToAllExcept(line, except string) {
	s.localUsers.Range(func(n, c interface{}) bool {
		name := n.(string)
		if name != except {
			conn := c.(*Conn)
			conn.SendLine(line)
		}
		return true
	})
}

func (s *Server) LookupSK(name string) (sk *crypto.SecretKey) {
	s.visit(name, func(conn *Conn) {
		sk = conn.sk
	})
	return
}

type lineEvent struct {
	fromNick string
	line     string
}

func (s *Server) s2slineLoop() {
	for s.Running() {
		lineEv := <-s.lineChnl
		log.Debugf("ircd server line from %s: %s", lineEv.fromNick, lineEv.line)
		pm := getPrivmsg(lineEv.line)
		if pm != nil {
			s.onPrivmsgFrom(pm, lineEv.fromNick)
		}
	}
}

func (s *Server) Running() bool {
	return !s.quit
}

func (s *Server) onPrivmsgFrom(pm *privMSG, from string) {
	if pm.Target.IsChan() {
		s.broadcastToAllExcept(pm.String(), from)
		if s.RemoteHandler != nil {
			s.RemoteHandler.SendBCMsg(pm.Target.String(), pm.Message)
		}
	} else if s.hasNick(pm.Target.String()) {
		s.visit(pm.Target.String(), func(c *Conn) {
			c.SendLine(pm.String())
		})
	} else if s.RemoteHandler != nil {
		if s.RemoteHandler.HasUser(pm.Target.String()) {
			s.RemoteHandler.SendPrivMsg(pm.Target.String(), pm.Message)
		} else {
			s.visit(from, func(c *Conn) {
				c.Numeric(s.Name, RPL_NOSUCHNICK, "No Such nickname/user")
			})
		}
	} else {
		s.visit(from, func(c *Conn) {
			c.Numeric(s.Name, RPL_NOSUCHNICK, "No Such nickname/user")
		})
	}
}

func (s *Server) QueueS2SLine(line string) {
	if len(line) > 0 {
		s.lineChnl <- lineEvent{
			line: line,
		}
	}
}

func (s *Server) queueS2SLine(line string, c *Conn) {
	if len(line) > 0 {
		s.lineChnl <- lineEvent{
			line:     line,
			fromNick: c.Nick(),
		}
	}
}

func (s *Server) addConn(c *Conn) {
}

func (s *Server) acceptConn(c *Conn) {
	err := c.blockingHandshake(s.Name)
	if err == nil {
		s.addConn(c)
		c.runChat(s.Name, s.queueS2SLine)
	} else {
		c.Quit(s.Name, "bad handshake")
	}
}

func (s *Server) Serve(l net.Listener) (err error) {
	for err == nil {
		var c net.Conn
		c, err = l.Accept()
		if err == nil {
			s.acceptConn(&Conn{
				c:        textproto.NewConn(c),
				sk:       crypto.KeyGen(),
				sendChnl: make(chan string, 8),
			})
		}
	}
	s.quit = true
	return
}
