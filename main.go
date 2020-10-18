package main

import (
	"os"
	"path/filepath"
	"strings"
	"github.com/majestrate/XD/cmd/rpc"
	"github.com/majestrate/XD/cmd/xd"
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
