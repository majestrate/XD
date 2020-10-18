# XD

I2P BitTorrent Client written in GO

![XD](contrib/logos/xd_logo_256x256.png)

[![Packaging status](https://repology.org/badge/vertical-allrepos/xd-torrent.svg)](https://repology.org/metapackage/xd-torrent)

![Downloads](https://img.shields.io/github/downloads/majestrate/XD/total.svg)

![MIT License](https://img.shields.io/github/license/majestrate/XD.svg)
![Logo is ebin](https://img.shields.io/badge/logo-ebin-brightgreen.svg)

## Features

Current:

* i2p only, no chances of cross network contamination, aka no way to leak IP.
* works with [i2pd](https://github.com/purplei2p/i2pd) and Java I2P using the SAM api
* Magnet URIs

Soon:

* transmission compatible RPC

Eventually:

* DHT Support
* Maggot Support

## Dependencies

* GNU Make
* GO 1.8 or higher


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

to compile XD to use [lokinet](https://github.com/loki-project/loki-network) by default use:

    $ make LOKINET=1

or use `go get`:

    $ go get -u -v -tags lokinet github.com/majestrate/XD

### cross compile for Raspberry PI

Set `GOARCH` and `GOOS` when building with make:

    $ make GOARCH=arm GOOS=linux

## Usage

To autogenerate a new config and start:

    $ ./XD torrents.ini

after started put torrent files into `./storage/downloads/` to start downloading

to seed torrents put data files into `./storage/downloads/` first then add torrent files

if you compiled with web ui it will be up at http://127.0.0.1:1776/

To use the RPC Tool symlink `XD` to `XD-CLI`

    $ ln -s XD XD-CLI

to list torrents run:

    $ ./XD-CLI list

to add a torrent from http server:

    $ ./XD-CLI add http://somehwere.i2p/some_torrent_that_is_not_fake.torrent

Optionally on non windows systems you can install XD to `/usr/local/`

    # make install

Or your home directory, make sure `$HOME/bin` is in your $PATH

    $ make install PREFIX=$HOME

