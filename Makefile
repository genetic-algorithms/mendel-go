SHELL ?= /bin/bash -e
BINARY ?= mendel-go
export VERSION ?= 1.0.0
export RELEASE ?= 5
# rpmbuild does not give us a good way to set topdir, so use the default location
RPMROOT ?= $(HOME)/rpmbuild
RPMNAME ?= mendel-go

# Make will search this relative paths for input and target files
VPATH=pprof

default: run

$(BINARY): mendel.go */*.go
	@echo GOOS=$$GOOS
	glide --quiet install
	go build -o $@

run: $(BINARY)
	time ./$? -f test/input/mendel-case1.ini

runlong: $(BINARY)
	time ./$? -f test/input/mendel-long.ini

cpu.pprof: run
	go tool pprof -text ./$(BINARY) ./pprof/$@

mem.pprof: run
	go tool pprof -text ./$(BINARY) ./pprof/$@

run-defaults: $(BINARY)
	time ./$(BINARY) -d

# Remember to up VERSION above. If building the rpm on mac, first: brew install rpm
# Note: during rpmbuild, get this benign msg: error: Couldn't exec /usr/local/Cellar/rpm/4.14.1_1/lib/rpm/elfdeps: No such file or directory
rpmbuild:
	mkdir -p $(RPMROOT)/{SOURCES,SRPMS,SRPMS}
	rm -f $(RPMNAME)-$(VERSION); ln -s . $(RPMNAME)-$(VERSION)  # so the tar file files can have this prefix
	tar --exclude '.git*' -X .tarignore -H -czf $(RPMROOT)/SOURCES/$(RPMNAME)-$(VERSION).tar.gz $(RPMNAME)-$(VERSION)
	rm -f $(RPMROOT)/SRPMS/$(RPMNAME)*rpm $(RPMROOT)/RPMS/x86_64/$(RPMNAME)*rpm
	GOOS=linux rpmbuild --target x86_64-linux -ba pkg/rpm/$(RPMNAME).spec
	rm -f $(RPMNAME)-$(VERSION)   # remove the sym link

test-main: mendel_test.go $(BINARY)
	glide --quiet install
	go test -v $<

test-pkgs:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default run prof run-defaults test-main test-pkgs clean
