// Dialer example.
//
// To the extent possible under law, Yawning Angel waived all copyright
// and related or neighboring rights to bulb, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package main

import (
	"io/ioutil"
	"log"
	"net/http"

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

	// Get a proxy.Dialer that will use the given Tor instance for outgoing
	// connections.
	dialer, err := c.Dialer(nil)
	if err != nil {
		log.Fatalf("Failed to get Dialer: %v", err)
	}

	// Try using the Dialer for something...
	orTransport := &http.Transport{Dial: dialer.Dial}
	orHTTPClient := &http.Client{Transport: orTransport}
	resp, err := orHTTPClient.Get("https://check.torproject.org/api/ip")
	if err != nil {
		log.Fatalf("Failed https GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("HTTPS GET via Tor: %v", resp)
	log.Printf(" Body: %s\n", body)
}
