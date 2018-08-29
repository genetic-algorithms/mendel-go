SHELL ?= /bin/bash -e
BINARY ?= mendel-go
export VERSION ?= 1.1.1
export RELEASE ?= 1
# rpmbuild does not give us a good way to set topdir, so use the default location
RPMROOT ?= $(HOME)/rpmbuild
RPMNAME ?= mendel-go
MAC_PKG_IDENTIFIER ?= com.github.genetic-algorithms.pkg.$(BINARY)
MAC_PKG_INSTALL_DIR ?= /Users/Shared/$(BINARY)

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
	rm -f $(RPMROOT)/SOURCES/$(RPMNAME)-*.tar.gz
	tar --exclude '.git*' -X .tarignore -H -czf $(RPMROOT)/SOURCES/$(RPMNAME)-$(VERSION).tar.gz $(RPMNAME)-$(VERSION)
	rm -rf $(RPMROOT)/BUILD/mendel-go-*
	rm -f $(RPMROOT)/SRPMS/$(RPMNAME)*rpm $(RPMROOT)/RPMS/x86_64/$(RPMNAME)*rpm $(RPMROOT)/RPMS/x86_64/$(RPMNAME)*rpm.gz
	GOOS=linux rpmbuild --target x86_64-linux -ba pkg/rpm/$(RPMNAME).spec
	gzip --keep $(RPMROOT)/RPMS/x86_64/$(RPMNAME)-$(VERSION)-$(RELEASE).x86_64.rpm
	rm -f $(RPMNAME)-$(VERSION)   # remove the sym link

# Remember to up VERSION above.
macpkg: $(BINARY)
	pkg/mac/populate-pkg-files.sh pkg/mac/$(BINARY)
	pkgbuild --root pkg/mac/$(BINARY) --scripts pkg/mac/scripts --identifier $(MAC_PKG_IDENTIFIER) --version $(VERSION) --install-location $(MAC_PKG_INSTALL_DIR) pkg/mac/build/$(BINARY)-$(VERSION).pkg
	rm -f pkg/mac/build/$(BINARY)-$(VERSION).pkg.zip
	cd pkg/mac/build; zip $(BINARY)-$(VERSION).pkg.zip $(BINARY)-$(VERSION).pkg; cd ../../..   # need to be in the same dir to zip

macinstall: macpkg
	sudo installer -pkg pkg/mac/build/$(BINARY)-$(VERSION).pkg -target '/Volumes/Macintosh HD'

macpkginfo:
	pkgutil --pkg-info $(MAC_PKG_IDENTIFIER)
	pkgutil --only-files --files $(MAC_PKG_IDENTIFIER)

upload-release:
	#TODO: create target for creating a release: https://developer.github.com/v3/repos/releases/#create-a-release

release: rpmbuild macpkg upload-release

test-main: mendel_test.go $(BINARY)
	glide --quiet install
	go test -v $<

test-pkgs:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default run prof run-defaults rpmbuild macpkg macinstall macpkginfo test-main test-pkgs clean
