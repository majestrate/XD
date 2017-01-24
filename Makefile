
xd: test
	GOPATH=$(PWD) go build -v xd/cmd/xd

test:
	GOPATH=$(PWD) go test -v ./...

clean:
	rm -f xd
