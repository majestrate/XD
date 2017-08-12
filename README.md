# XD

Standalone I2P BitTorrent Client written in GO

![XD](contrib/logos/xd_logo_256x256.png)

## Features

Current:

* i2p only, no chances of cross network contamination, aka no way to leak IP.
* no java required, works with [i2pd](https://github.com/purplei2p/i2pd)

Soon:

* DHT/Magnet Support

Eventually:

* Maggot Support (?)
* rtorrent compatible RPC (?)

## Dependencies

* go 1.3 **or** go 1.8
* GNU Make

## Building


You can either use `make` or `go get` to build XD

the `make` way (preferred):

    $ git clone https://github.com/majestrate/XD
    $ cd XD
    $ make

the `go get` way, requires go 1.6 or higher because it uses vendored packages

    $ go get -u -v github.com/majestrate/XD

### cross compile for Raspberry PI

Set `GOARCH` and `GOOS` when building with make:

    $ make GOARCH=arm GOOS=linux


## Usage

To autogenerate a new config and start:

    $ ./XD torrents.ini

after started put torrent files into `./storage/downloads/` to start downloading

to seed torrents put data files into `./storage/downloads/` first then add torrent files

To use the RPC Tool symlink `XD` to `XD-CLI`

    $ ln -s XD XD-CLI

to list torrents run:

    $ ./XD-CLI


to add a torrent from http server:

    $ ./XD-CLI add http://somehwere.i2p/some_torrent_that_is_not_fake.torrent
