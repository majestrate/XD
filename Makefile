GOPATH=$(PWD)

xd:
	go build -v xd/cmd/xd

test:
	go build -v ./...
	go test -v ./...

clean:
	rm -f xd
