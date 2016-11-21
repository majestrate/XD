GOPATH=$(PWD)

xd:
	go build -v xd/cmd/xd

clean:
	rm -f xd
