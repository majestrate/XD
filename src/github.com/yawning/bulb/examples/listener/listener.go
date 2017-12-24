// Listener example.
//
// To the extent possible under law, Yawning Angel waived all copyright
// and related or neighboring rights to bulb, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log"
	"net/http"

	"github.com/yawning/bulb"
	"github.com/yawning/bulb/utils/pkcs1"
)

func onionServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, onion world!\n")
}

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

	// Generate a private key and create a port 80 Onion Service.
	//
	// For one-shot services:` l, err := c.Listener(80, nil)` is considerably
	// easier.
	pk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatalf("Failed to generate RSA key")
	}
	id, err := pkcs1.OnionAddr(&pk.PublicKey)
	if err != nil {
		log.Fatalf("Failed to derive onion ID: %v", err)
	}
	log.Printf("Expected ID: %v", id)

	cfg := &bulb.NewOnionConfig{
		DiscardPK: true,
		PrivateKey: pk,
	}
	l, err := c.NewListener(cfg, 80)
	if err != nil {
		log.Fatalf("Failed to get Listener: %v", err)
	}
	defer l.Close()

	log.Printf("Listener: %s", l.Addr())
	http.HandleFunc("/", onionServer)
	http.Serve(l, nil)
}
