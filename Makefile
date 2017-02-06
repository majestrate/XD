
xd:
	GOPATH=$(PWD) go build -v

test:
	GOPATH=$(PWD) go test -v xd/...

clean:
	rm -f xd
