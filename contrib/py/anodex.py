#!/usr/bin/env python3
#
# anodex mirroring script for transmission to XD
#
# takes all horriblesubs torrents from transmission and
# uploads them to anodex.i2p then adds them to XD for seeding
#
# uses ~/.netrc for authentication, see man 5 netrc
# example entry:
# machine anodex.i2p username youruser password yourpassword
#
# usage:
# python3 -m venv v && v/bin/pip install -r anodex_requirements.txt
# v/bin/python anodex.py

import transmissionrpc as rpc
import requests
import shutil
import sys
import os


# http proxy url
proxy_url = "http://127.0.0.1:4444/"

# xd downloads directory
xd_dir = "/home/xd/storage/downloads/"

# xd api endpoint
xd_api = "http://127.0.0.1:1488/ecksdee/api"

# anodex info
anodexBaseURL = "http://anodex.i2p"
anodexAnimeURL = "{}/c/3/?t=json".format(anodexBaseURL)

# proxies for i2p
proxies = {
    "http": proxy_url,
    "https": proxy_url
}

# transmission parameters
rpc_host = '127.0.0.1'
rpc_port = 9091
rpc_user = None
rpc_password = None


def anodex_upload_torrent(torrent, tags, description='auto upload'):
    """
    upload a torrent to andoex anime category
    returns torrent url
    """
    j = None
    print("upload {}".format(torrent.name))
    with open(torrent.torrentFile, 'rb') as f:
        r = requests.post(anodexAnimeURL, files={
            'torrent-file' : (os.path.basename(torrent.torrentFile), f, 'application/bittorrent'),
        }, data={
            "torrent-name": torrent.name,
            "torrent-description": description,
            "tags": ','.join(tags)
        }, proxies=proxies)
        j = r.json()
    if j and 'URL' in j:
        return j['URL']
        

def copy_torrent_files(torrent, dest_dir):
    """
    copy torrent data files from torrent into dest_dir
    """
    files = torrent.files()
    for id in files:
        inf = os.path.join(torrent.downloadDir, files[id]['name'])
        outf = os.path.join(dest_dir, files[id]['name'])
        d = os.path.dirname(outf)
        if not os.path.exists(d):
            os.mkdir(d)
        if os.path.exists(inf):
            print("{} -> {}".format(inf, outf))
            shutil.copyfile(inf, outf)
    
def anodex_has_torrent(torrent):
    """
    return true if anodex has a copy of this torrent
    """
    r = requests.get("{}/dl/{}.torrent".format(anodexBaseURL, torrent.hashString), proxies=proxies)
    return r.status_code is 200


def xd_has_torrent(torrent):
    """
    return true if xd has this torrent already
    """
    r = requests.post(xd_api, json={'method': 'XD.TorrentStatus', 'infohash': torrent.hashString})
    j = r.json()
    return 'error' not in j or j['error'] is None

def should_process(torrent):
    """
    return true if we should process this torrent
    """
    # only download torrents that have all the data
    if not torrent.isFinished:
        return False
    if xd_has_torrent(torrent):
        return False
    return torrent.name.lower().startswith("[horriblesubs]")

def xd_add_torrent(url):
    """
    add torrent to xd by url
    """
    print('adding {}'.format(url))
    r = requests.post(xd_api, json={'method': 'XD.AddTorrent', 'url': url})
    j = r.json()


def generate_tags(t):
    """
    given a torrent generate anodex tags
    """
    tags = list()
    name = t.name.lower()
    if name.startswith('[horriblesubs]'):
        tags.append('horriblesubs')
    if '[1080p]' in name:
        tags.append('1080p')
    elif '[720p]' in name:
        tags.append('720p')
    return tags

def main():
    cl = rpc.Client(rpc_host, port=rpc_port, user=rpc_user, password=rpc_password)
    torrents = cl.get_torrents()
    for t in torrents:
        if should_process(t):
            copy_torrent_files(t, xd_dir)
            if not anodex_has_torrent(t):
                tries = 10
                while tries > 0:
                    u = anodex_upload_torrent(t, generate_tags(t))
                    if u:
                        print('uploaded to {}'.format(u))
                        break
                    else:
                        print('upload failed, try again, {} tries left'.format(tries))
                        tries -= 1
            url = '{}/dl/{}.torrent'.format(anodexBaseURL, t.hashString)
            xd_add_torrent(url)


if __name__ == "__main__":
    main()
