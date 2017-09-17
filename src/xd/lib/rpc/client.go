package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"xd/lib/bittorrent/swarm"
)

type Client struct {
	url     string
	swarmno string
}

func NewClient(url string, swarmno int) *Client {
	return &Client{
		url:     url,
		swarmno: fmt.Sprintf("%d", swarmno),
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
	err = cl.doRPC(&ListTorrentsRequest{BaseRequest{cl.swarmno}}, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&torrents)
	})
	return
}

func (cl *Client) SetPieceWindow(n int) (err error) {
	err = cl.doRPC(&SetPieceWindowRequest{BaseRequest{cl.swarmno}, n}, func(r io.Reader) error {
		var response interface{}
		return json.NewDecoder(r).Decode(&response)
	})
	return
}

func (cl *Client) AddTorrent(url string) (err error) {
	err = cl.doRPC(&AddTorrentRequest{BaseRequest{cl.swarmno}, url}, func(r io.Reader) error {
		var response interface{}
		return json.NewDecoder(r).Decode(&response)
	})
	return
}

func (cl *Client) SwarmStatus(ih string) (st swarm.TorrentStatus, err error) {
	err = cl.doRPC(&TorrentStatusRequest{BaseRequest{cl.swarmno}, ih}, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&st)
	})
	return
}
