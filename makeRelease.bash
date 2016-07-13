#!/usr/bin/env bash

function build () {
    echo "Building for $1_$2..."
    os=$1;shift
    arch=$1;shift
    make GOOS=$os GOARCH=$arch $@ all
}

build solaris amd64 $@
build linux amd64 $@
build darwin amd64 $@

