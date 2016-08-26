#/usr/bin/env bash
# Helper script for building inside docker container to keep the Makefile shorter
# See makefile target `dockerbuild`

cd $1

go get -u github.com/jteeuwen/go-bindata/...

make BUILD_SUFFIX=docker

