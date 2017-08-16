REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))


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

$(XD):
	$(GO) build -v -ldflags "-X xd/lib/version.Git=-$(shell git rev-parse --short HEAD)" -o $(XD)

test:
	$(GO) test -v xd/...

clean:
	$(GO) clean -v
