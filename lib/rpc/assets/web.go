//go:build webui
// +build webui

package assets

import (
	"embed"
	"net/http"
)

// content holds our static web server content.
//
//go:embed favicon.png
//go:embed xd.min.js
//go:embed xd.css
//go:embed index.html
var content embed.FS

func GetAssets() http.FileSystem {
	return http.FS(content)
}
