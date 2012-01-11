
include $(GOROOT)/src/Make.inc

all:	src/cmd/_obj/sid.$O src/pkg/sid/_obj/sid.a
	mkdir -p bin/$(GOARCH)
	$(LD) -Lsrc/pkg/sid/_obj -o bin/$(GOARCH)/sid src/cmd/_obj/sid.$O

src/cmd/_obj/sid.$O:	src/cmd/sid.go
	$(MAKE) -C src/cmd

src/pkg/sid/_obj/sid.a: \
	src/pkg/sid/config.go \
	src/pkg/sid/control.go \
	src/pkg/sid/doc.go \
	src/pkg/sid/http.go
	$(MAKE) -C src/pkg/sid

clean:
	$(MAKE) -C src/cmd clean
	$(MAKE) -C src/pkg/sid clean

install:
	@echo "No automatic installation -- refer to README.mkd"

test:
	@echo "No tests available."

bench:
	@echo "No benchmark runs available."
	
nuke:	clean
	rm -rf bin
	rm -rf src/pkg/sid/_obj
	rm -rf src/pkg/sid/_obj
