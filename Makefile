GO ?= $(shell which go)
GOFMT ?= $(shell which gofmt)
GREP ?= $(shell which grep)
GOOS ?= $(shell uname | tr A-Z a-z)
GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILD_SUFFIX = $(GOOS)_$(GOARCH)

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


GO_PACKAGES := msp master slave model notifier

.PHONY: check-format
check-format:
	@! $(GOFMT) -d $(GO_PACKAGES) | $(GREP) '^'

.PHONY: format
format:
	@ $(GOFMT) -w $(GO_PACKAGES)

