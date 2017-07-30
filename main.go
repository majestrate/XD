package main

import (
	"os"
	"strings"
	"xd/cmd/rpc"
	"xd/cmd/xd"
)

func main() {
	exename := strings.ToUpper(os.Args[0])
	docli := exename == "XD-CLI" || exename == "XD-CLI.EXE"
	if docli {
		rpc.Run()
	} else {
		xd.Run()
	}
}
