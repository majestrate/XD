GOPATH=$(PWD)

xd:
	go build -v xd/cmd/xd

test:
	go test -v ./...

clean:
	rm -f xd
