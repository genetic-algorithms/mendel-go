SHELL = /bin/bash -e
BINARY = mendel-go

# Make will search this relative paths for input and target files
VPATH=pprof

default: run

$(BINARY): mendel.go */*.go
	glide --quiet install
	go build

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

test-main: mendel_test.go $(BINARY)
	glide --quiet install
	go test -v $<

test-pkgs:
	glide --quiet install
	go test ./random

clean:
	go clean

.PHONY: default run prof run-defaults test-main test-pkgs clean
