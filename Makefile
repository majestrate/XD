REPO = $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

build:
	GOPATH=$(REPO) go build -v xd

test:
	GOPATH=$(REPO) go test -v xd/...

test-storage:
	GOPATH=$(REPO) go test -v xd/lib/storage

rpc:
	GOPATH=$(REPO) go build -v xd/cmd/rpcdebug

clean:
	GOPATH=$(REPO) go clean -v xd/...
