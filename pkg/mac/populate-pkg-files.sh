#!/bin/bash

# Copy the files that should be packaged into the mac .pkg file.
# It is assumed this script will be run in the top dir of the mendel-go repo, and the mendel-go binary has already been built.

BUILD_ROOT="$1"
if [[ ! -d "$BUILD_ROOT" ]]; then
	echo "Usage: $0 <root-dir>"
	exit 1
fi

mkdir -p $BUILD_ROOT/bin $BUILD_ROOT/share/mendel-go
cp mendel-go $BUILD_ROOT/bin
cp mendel-defaults.ini LICENSE COPYRIGHT $BUILD_ROOT/share/mendel-go
cp test/input/mendel-short.ini $BUILD_ROOT/share/mendel-go
