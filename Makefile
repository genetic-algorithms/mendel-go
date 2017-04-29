SHELL = /bin/bash -e
BINARY = mendel-go

default: run

$(BINARY): mendel.go */*.go
	glide --quiet install
	go build

run: $(BINARY)
	time ./$(BINARY) -f test/input/mendel-case1.ini

run-defaults: $(BINARY)
	time ./$(BINARY) -d

test-main: mendel_test.go $(BINARY)
	glide --quiet install
	go test -v mendel_test.go

test-pkgs:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default run run-defaults test-main test-pkgs clean
