
xd: test
	GOPATH=$(PWD) go build -v

test:
	GOPATH=$(PWD) go test -v

clean:
	rm -f xd
