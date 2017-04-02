SHELL = /bin/bash -e

default: build run

build:
	glide --quiet install
	go build

run:
	time ./mendel-go -f test/input/mendel-case1.ini

test:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default build run test clean
