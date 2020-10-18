package rpc

import (
	"io"
)

type rpcIO struct {
	w io.Writer
	r io.ReadCloser
}

func (r *rpcIO) Close() error {
	return r.r.Close()
}

func (r *rpcIO) Write(d []byte) (int, error) {
	return r.w.Write(d)
}

func (r *rpcIO) Read(d []byte) (int, error) {
	return r.r.Read(d)
}
