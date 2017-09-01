# Getting started

Once you have built or obtained a release, a webui will be enabled by default at http://127.0.0.1:1488/

Windows users are encouraged to use the webui if they can't use the command line tool

## Command Line (unix)

To use the command line tool you must symlink the `XD` binary to `XD-cli`

    $ ln -s XD XD-cli

Adding torrents from seed file over i2p:

    XD-cli add http://somesite.i2p/some/url/to/a/torrent.torrent

Listing active torrents:

    XD-cli list

To increase how many pieces to request in parallel use `set-piece-window` command (may be removed in future):

    XD-cli set-piece-window 10


## Command Line (windows)

On Windows: make a copy of the file called `XD-cli.exe`

All commands are done in the same manner as in unix, except `/` needs escaping depending on what terminal in use.

TODO: add more docs for windows
