package rpc

import (
	"encoding/json"
	"net/http"
)

type ResponseWriter struct {
	w http.ResponseWriter
}

func (rw *ResponseWriter) SendJSON(obj interface{}) {
	json.NewEncoder(rw.w).Encode(obj)
}

func (rw *ResponseWriter) SendError(msg string) {
	rw.SendJSON(map[string]string{
		"error": msg,
	})
}

func (rw *ResponseWriter) Return(obj interface{}) {
	rw.SendJSON(obj)
	/*
		rw.SendJSON(map[string]interface{}{
			"error":  nil,
			"result": obj,
		})
	*/
}
