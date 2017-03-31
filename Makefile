REPO = $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

build: clean XD rpc

XD:
	GOPATH=$(REPO) go build -v -ldflags "-X xd/lib/version.Git=-$(shell git rev-parse --short HEAD)"

test:
	GOPATH=$(REPO) go test -v xd/...

test-storage:
	GOPATH=$(REPO) go test -v xd/lib/storage

rpc: rpcdebug

rpcdebug:
	GOPATH=$(REPO) go build -v xd/cmd/rpcdebug

storetest:
	GOPATH=$(REPO) go build -v xd/cmd/storetest


clean:
	GOPATH=$(REPO) go clean -v
	rm -f storetest rpcdebug
