
xd:
	GOPATH=$(PWD) go build -v

test:
	GOPATH=$(PWD) go test -v xd/...

test-storage:
	GOPATH=$(PWD) go test -v xd/lib/storage



clean:
	rm -f xd
