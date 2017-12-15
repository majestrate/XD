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

* GNU Make
* GO 1.9


## Building

right now the best way to build is with `make`

    $ git clone https://github.com/majestrate/XD
    $ cd XD
    $ make

if you do not want to build with embedded webui instead run:

    $ make no-webui

you can build with go get using:

    $ go get -u -v github.com/majestrate/XD

please note that using `go get` disables the webui.

### cross compile for Raspberry PI

Set `GOARCH` and `GOOS` when building with make:

    $ make GOARCH=arm GOOS=linux

## Usage

To autogenerate a new config and start:

    $ ./XD torrents.ini

after started put torrent files into `./storage/downloads/` to start downloading

to seed torrents put data files into `./storage/downloads/` first then add torrent files

if you compiled with web ui it will be up at http://127.0.0.1:1488/

To use the RPC Tool symlink `XD` to `XD-CLI`

    $ ln -s XD XD-CLI

to list torrents run:

    $ ./XD-CLI list

to add a torrent from http server:

    $ ./XD-CLI add http://somehwere.i2p/some_torrent_that_is_not_fake.torrent
