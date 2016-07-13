GO ?= $(shell which go)
GOOS ?= $(shell uname | tr A-Z a-z)
GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILD_SUFFIX = $(GOOS)_$(GOARCH)

.PHONY: all
all: master/bin/master_$(BUILD_SUFFIX) slave/bin/slave_$(BUILD_SUFFIX)


.PHONY: clean
clean: clean_master clean_slave


master/bin/master_$(BUILD_SUFFIX):
	cd master && $(GO) build -o bin/master_$(BUILD_SUFFIX) master.go

.PHONY:clean_master
clean_master:
	cd master/ && $(GO) clean
	rm -rf master/bin


slave/bin/slave_$(BUILD_SUFFIX):
	cd slave && $(GO) build -o bin/slave_$(BUILD_SUFFIX) slave.go controller.go

.PHONY: clean_slave
clean_slave:
	cd slave/ && $(GO) clean
	rm -rf slave/bin
