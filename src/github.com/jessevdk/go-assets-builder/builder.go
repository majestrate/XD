package main

import (
	"github.com/jessevdk/go-assets"
	"github.com/jessevdk/go-flags"
	"os"
)

func main() {
	var opts struct {
		PackageName  string `short:"p" long:"package" description:"The package name to generate the assets for" default:"main"`
		VariableName string `short:"v" long:"variable" description:"The name of the generated asset tree" default:"Assets"`
		StripPrefix  string `short:"s" long:"strip-prefix" description:"Strip the specified prefix from all paths"`
		Output       string `short:"o" long:"output" description:"File to write output to, or - to write to stdout" default:"-"`
	}

	p := flags.NewParser(&opts, flags.Default)
	p.Usage = "[OPTIONS] FILES..."

	args, err := p.Parse()

	if err != nil {
		os.Exit(1)
	}

	g := assets.Generator{
		PackageName:  opts.PackageName,
		VariableName: opts.VariableName,
		StripPrefix:  opts.StripPrefix,
	}

	for _, f := range args {
		if err := g.Add(f); err != nil {
			panic(err)
		}
	}

	if len(opts.Output) == 0 || opts.Output == "-" {
		g.Write(os.Stdout)
	} else {
		f, err := os.Create(opts.Output)

		if err != nil {
			panic(err)
		}

		defer f.Close()

		if err := g.Write(f); err != nil {
			panic(err)
		}
	}
}
