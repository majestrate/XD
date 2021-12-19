package main

import (
	"github.com/majestrate/XD/cmd/rpc"
	"github.com/majestrate/XD/cmd/xd"
	"os"
	"path/filepath"
	"strings"
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
