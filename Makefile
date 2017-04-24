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

test:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default run test clean
