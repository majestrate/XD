package util

import (
	"net/http"
)

func GET(mux *http.ServeMux, path string, h http.HandlerFunc) {
	methodOnly(mux, "GET", path, h)
}

func PUT(mux *http.ServeMux, path string, h http.HandlerFunc) {
	methodOnly(mux, "PUT", path, h)
}

func POST(mux *http.ServeMux, path string, h http.HandlerFunc) {
	methodOnly(mux, "POST", path, h)
}

func DELETE(mux *http.ServeMux, path string, h http.HandlerFunc) {
	methodOnly(mux, "DELETE", path, h)
}

func methodOnly(mux *http.ServeMux, method, path string, h http.HandlerFunc) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method {
			h(w, r)
		} else {
			status := http.StatusMethodNotAllowed
			http.Error(w, http.StatusText(status), status)
		}
	})
}
