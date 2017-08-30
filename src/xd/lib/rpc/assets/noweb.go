// +build no_webui

package assets

import (
	"net/http"
)

func GetAssets() http.FileSystem {
	return nil
}
