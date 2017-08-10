package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"xd/lib/bittorrent/swarm"
)

type Client struct {
	url string
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

func (cl *Client) doRPC(r interface{}, h func(r io.Reader) error) (err error) {
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(r)
	if err == nil {
		var resp *http.Response
		resp, err = http.Post(cl.url, RPCContentType, &buf)
		if err == nil {
			err = h(resp.Body)
			resp.Body.Close()
		}
	}
	return
}

func (cl *Client) ListTorrents() (torrents swarm.TorrentsList, err error) {
	err = cl.doRPC(&ListTorrentsRequest{}, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&torrents)
	})
	return
}

func (cl *Client) SwarmStatus(ih string) (st swarm.TorrentStatus, err error) {
	err = cl.doRPC(&TorrentStatusRequest{Infohash: ih}, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&st)
	})
	return
}
