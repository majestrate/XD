# go-assets-builder
go-assets-builder is a simple asset builder program to generate embedded assets
using [go-assets](https://github.com/jessevdk/go-assets). This builder program
exposes the various [go-assets](https://github.com/jessevdk/go-assets) generator
options as command line options and allows for convenient generation of
go embedded assets from for example a Makefile.

```console
Usage:
  go-assets-builder [OPTIONS] FILES...

Help Options:
  -h, --help=         Show this help message

Application Options:
  -p, --package=      The package name to generate the assets for (main)
  -v, --variable=     The name of the generated asset tree (Assets)
  -s, --strip-prefix= Strip the specified prefix from all paths
  -c, --compressed    Enable gzip compression of assets
  -o, --output=       File to write output to, or - to write to stdout (-)
```
