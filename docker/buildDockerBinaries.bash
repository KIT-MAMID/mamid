#/usr/bin/env bash
# Helper script for building inside docker container to keep the Makefile shorter
# See makefile target `dockerbuild`

cd $1
pushd vendor/github.com/mattn/go-sqlite3
go install
popd
go get -t ./...

make BUILD_SUFFIX=docker

