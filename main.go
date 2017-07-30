package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"xd/cmd/rpc"
	"xd/cmd/xd"
)

func main() {
	exename := filepath.Base(strings.ToUpper(os.Args[0]))
	docli := exename == "XD-CLI" || exename == "XD-CLI.EXE"
	if docli {
		rpc.Run()
	} else {
		xd.Run()
	}
}
