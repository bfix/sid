
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
	@echo GOPATH=$(GOPATH)
	$(MAKE) -C sid install
	$(MAKE) -C dcd install
	$(GOINST)

test:
	GOPATH=${BASE} $(MAKE) -C sid test
	GOPATH=${BASE} $(MAKE) -C dcd test

clean:
	GOPATH=${BASE} $(MAKE) -C sid clean
	GOPATH=${BASE} $(MAKE) -C dcd clean
	rm -rf $(BASE)/pkg
	$(GOCLEAN)
