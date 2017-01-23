
xd: test
	GOPATH=$(PWD) go build -v xd/cmd/xd

test:
	GOPATH=$(PWD) go test ./...

clean:
	rm -f xd
