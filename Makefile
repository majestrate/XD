GOPATH=$(PWD)

xd: build
	go build -v xd/cmd/xd

build:
	go build -v ./...
test:
	go test -v ./...

clean:
	rm -f xd
