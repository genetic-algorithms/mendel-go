SHELL = /bin/bash -e

default: build run

build:
	glide --quiet install
	go build

run:
	./mendel-go

test:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default build run test clean
