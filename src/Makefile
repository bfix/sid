
PWD := $(shell pwd)
BASE := $(shell dirname $(PWD))

ifneq ($(shell basename $(PWD)), src)
$(error Not in source directory - you are here: '${PWD}')
endif

ifneq ($(strip $(GOPATH)),)
GOPATH := ${GOPATH}:$(BASE)
else
GOPATH := $(BASE)
endif

install:	fmt
	GOPATH=$(GOPATH) go install -v ./...

test:
	go test ./...

clean:
	go clean -i ./...
	
fmt:
	go fmt ./...
