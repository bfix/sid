
PWD := $(shell pwd)
BASE := $(shell dirname $(PWD))

ifneq ($(shell basename $(PWD)), src)
$(error Not in source directory - you are here: '${PWD}')
endif

install:	fmt
	go build -o ../bin/dcd dcd/dcd.go
	GOPATH=${PWD}/..:${GOPATH} go install sid

test:
	go test ./...

clean:
	go clean -i ./...
	
fmt:
	go fmt ./...
