SHELL = /bin/bash -e

# DOCKER_TAG ?= bld
# OS := $(shell uname)
# ifeq ($(OS),Darwin)
	# Mac OS X 
	# FOO ?= bar1
# else
	# Assume Linux (could test by test if OS is Linux)
	# FOO ?= bar2
# endif

default: mendel runmendel

%: %.go
	go build $<
	# -./$@

runmendel:
	./mendel

clean:
	rm -f mendel

.SECONDARY:

.PHONY: default clean
