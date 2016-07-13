GO ?= $(shell which go)
GOFMT ?= $(shell which gofmt)
GREP ?= $(shell which grep)
GOOS ?= $(shell uname | tr A-Z a-z)
GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILD_SUFFIX = $(GOOS)_$(GOARCH)

pkgs          = $(shell $(GO) list ./... | grep -v /vendor/)
pkg_dirs      = $(addprefix $(GOPATH)/src/,$(pkgs))

.PHONY: all
all: build/master_$(BUILD_SUFFIX) build/slave_$(BUILD_SUFFIX)


.PHONY: clean
clean: clean_master clean_slave


build/master_$(BUILD_SUFFIX):
	cd master && $(GO) build -o ../build/master_$(BUILD_SUFFIX) master.go

.PHONY:clean_master
clean_master:
	cd master/ && $(GO) clean
	rm -rf build/master*


build/slave_$(BUILD_SUFFIX):
	cd slave && $(GO) build -o ../build/slave_$(BUILD_SUFFIX) slave.go controller.go

.PHONY: clean_slave
clean_slave:
	cd slave/ && $(GO) clean
	rm -rf build/slave*

.PHONY: test
test:
	@$(GO) test -short $(pkgs)


.PHONY: check-format
check-format:
	@! $(GOFMT) -d $(pkg_dirs) | $(GREP) '^'

.PHONY: format
format:
	@ $(GOFMT) -w $(pkg_dirs)

.PHONY: vet
vet:
	@ $(GO) vet $(pkgs)

.PHONY: release
release:
	./makeRelease.bash

