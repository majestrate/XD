REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
LOGOS = $(REPO)/contrib/logos
WEBUI = ./contrib/webui
WEBUI_LOGO = $(WEBUI)/docroot/favicon.png
GO_ASSETS = $(REPO)/build-assets
WEB_FILES = $(WEBUI)/docroot/*


ifdef GOROOT
	GO = $(GOROOT)/bin/go
endif

GO ?= $(shell which go)

ifeq ($(GOOS),windows)
	XD := $(REPO)/XD.exe
else
	XD := $(REPO)/XD
endif

GOPATH := $(REPO)

build: clean $(XD)


$(GO_ASSETS):
	$(GO) build  -o $(GO_ASSETS) -v github.com/jessevdk/go-assets-builder

assets: $(GO_ASSETS)
	$(GO_ASSETS) -p assets  $(WEB_FILES) > $(REPO)/src/xd/lib/rpc/assets/assets.go

$(XD): assets
	$(GO) build -v -ldflags "-X xd/lib/version.Git=-$(shell git rev-parse --short HEAD)" -o $(XD)

test:
	$(GO) test -v xd/...

clean:
	$(GO) clean -v


$(WEBUI_LOGO):
	cp $(LOGOS)/xd_logo.png $(WEBUI_LOGO)

webui: $(WEBUI_LOGO)
	$(MAKE) -C $(WEBUI) clean build

run-webui:
	$(MAKE) -C $(WEBUI) run
