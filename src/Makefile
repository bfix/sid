
PWD := $(shell pwd)
BASE := $(shell dirname $(PWD))

ifneq ($(shell basename $(PWD)), src)
$(error Not in source directory - you are here: '${PWD}')
endif

ifneq ($(strip $(GOPATH)),)
GOPKG := $(shell echo ${GOPATH} | cut -f 1 -d ':')
GOPATH := ${GOPATH}:$(BASE)
GOINST := cp -vr $(BASE)/pkg/* $(GOPKG)/pkg; cp -vr $(BASE)/bin/* $(GOPKG)/bin
GOCLEAN := 	rm -f ${GOPKG}/pkg/${GOOS}_${GOARCH}/sid.a
else
GOPATH := $(BASE)
GOPKG := $(BASE)
GOINST := 
GOCLEAN := 
endif

install:
	echo $(GOPATH)
	go install ./...
	$(GOINST)

test:
	go test ./...

clean:
	GOPATH=${BASE} go clean ./...
	rm -rf $(BASE)/pkg
	$(GOCLEAN)
