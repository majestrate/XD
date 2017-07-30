package rpc

import (
	"encoding/json"
	"net/http"
)

type ResponseWriter struct {
	hrw http.ResponseWriter
}

func (rw *ResponseWriter) SendJSON(obj interface{}) {
	json.NewEncoder(rw.hrw).Encode(obj)
}

func (rw *ResponseWriter) SendError(msg string) {
	rw.SendJSON(map[string]string{
		"error": msg,
	})
}

func (rw *ResponseWriter) Return(obj interface{}) {
	rw.SendJSON(map[string]interface{}{
		"error":  nil,
		"result": obj,
	})
}
