# XD

Standalone I2P BitTorrent Client written in GO

## Features

Current:

* i2p only, no chances of cross network contamination, aka no way to leak IP.
* no java required, works with [i2pd](https://github.com/purplei2p/i2pd)

Soon:

* Make downloads store files properly [see issue #1](https://github.com/majestrate/XD/issues/1)
* DHT/Magnet Support
* Some Slick Logo thing for propaganda purposes

Eventually:

* Maggot Support (?)
* rtorrent compatible RPC (?)



## building

You can either use `make` or `go get` to build XD

the `make` way (preferred):

    $ git clone https://github.com/majestrate/XD
    $ cd XD
    $ make

the `go get` way, requires `$GOPATH` to be set properly

    $ go get -u -v github.com/majestrate/XD

## Usage

To autogenerate a new config and start:

    $ ./XD torrents.ini

after started put torrent files into `./storage/downloads/` to start downloading

seeding coming soon (tm)
