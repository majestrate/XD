REPO = $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

XD:
	GOPATH=$(REPO) go build -v -ldflags "-X xd/lib/version.Git=-$(shell git rev-parse --short HEAD)"

test:
	GOPATH=$(REPO) go test -v xd/...

test-storage:
	GOPATH=$(REPO) go test -v xd/lib/storage

rpc:
	GOPATH=$(REPO) go build -v xd/cmd/rpcdebug

clean:
	GOPATH=$(REPO) go clean -v
