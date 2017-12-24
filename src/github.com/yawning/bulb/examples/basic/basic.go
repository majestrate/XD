// Basic example.
//
// To the extent possible under law, Yawning Angel waived all copyright
// and related or neighboring rights to bulb, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package main

import (
	"log"

	"github.com/yawning/bulb"
)

func main() {
	// Connect to a running tor instance.
	//  * TCP: c, err := bulb.Dial("tcp4", "127.0.0.1:9051")
	c, err := bulb.Dial("unix", "/var/run/tor/control")
	if err != nil {
		log.Fatalf("failed to connect to control port: %v", err)
	}
	defer c.Close()

	// See what's really going on under the hood.
	// Do not enable in production.
	c.Debug(true)

	// Authenticate with the control port.  The password argument
	// here can be "" if no password is set (CookieAuth, no auth).
	if err := c.Authenticate("ExamplePassword"); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// At this point, c.Request() can be used to issue requests.
	resp, err := c.Request("GETINFO version")
	if err != nil {
		log.Fatalf("GETINFO version failed: %v", err)
	}
	log.Printf("GETINFO version: %v", resp)

	// If you want to use events, then you need to start up the async reader,
	// which demultiplexes responses and events.
	c.StartAsyncReader()

	// For example, watch circuit events till the app is killed.
	if _, err := c.Request("SETEVENTS CIRC"); err != nil {
		log.Fatalf("SETEVENTS CIRC failed: %v", err)
	}
	for {
		ev, err := c.NextEvent()
		if err != nil {
			log.Fatalf("NextEvent() failed: %v", err)
		}
		log.Printf("Circuit event: %v", ev)
	}
}
