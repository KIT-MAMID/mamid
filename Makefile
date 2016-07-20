# User-settable
GO 			?= $(shell which go)
GOFMT 			?= $(shell which gofmt)
GREP 			?= $(shell which grep)
GOOS 			?= $(shell uname | tr A-Z a-z)
TESTBED_SLAVE_COUNT 	?= 3
SUDO 			?= $(shell if [[ $(GOOS) != "darwin" ]]; then which sudo; fi)

########################################################################################################################

## Commands
GOFILES_IN_DIR_CMD = $(shell find $(1) -type f \( -iname '*.go' ! -iname '.*' \))
GOFILES_IN_DIRS = $(foreach dir,$(1),$(call GOFILES_IN_DIR_CMD,$(dir)))

## Onetimers
GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILD_SUFFIX  ?= $(GOOS)_$(GOARCH)
pkgs          = $(shell $(GO) list ./... | grep -v /vendor/)
pkg_dirs      = $(addprefix $(GOPATH)/src/,$(pkgs))

########################################################################################################################

.PHONY: all
all: build/master_$(BUILD_SUFFIX) build/slave_$(BUILD_SUFFIX) build/notifier_$(BUILD_SUFFIX)

.PHONY: clean
clean: clean_master clean_slave clean_testbed

########################################################################################################################

build/master_$(BUILD_SUFFIX): $(call GOFILES_IN_DIRS,master/ msp/ model/)
	cd master/cmd && $(GO) build -o ../../build/master_$(BUILD_SUFFIX)

.PHONY:clean_master
clean_master:
	cd master/ && $(GO) clean
	rm -rf build/master*

########################################################################################################################

build/notifier_$(BUILD_SUFFIX): $(call GOFILES_IN_DIRS,notifier/)
	cd notifier && $(GO) build -o ../build/notifier_$(BUILD_SUFFIX)

.PHONY:clean_notifier
clean_notifier:
	cd notifier/ && $(GO) clean
	rm -rf build/notifier*

########################################################################################################################

build/slave_$(BUILD_SUFFIX): $(call GOFILES_IN_DIRS,slave/ msp/)
	cd slave/cmd && $(GO) build -o ../../build/slave_$(BUILD_SUFFIX)

.PHONY: clean_slave
clean_slave:
	cd slave/ && $(GO) clean
	rm -rf build/slave*

########################################################################################################################

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

########################################################################################################################

# The docker-based local staging environment

.PHONY: testbed_up testbed_down testbed_net clean_testbed dockerbuild

clean_testbed: testbed_down
	rm -f docker/*.depend
	rm -rf docker/.dockergopath
	-$(SUDO) docker rmi mamid/builder
	-$(SUDO) docker rmi mamid/master
	-$(SUDO) docker rmi mamid/slave
	-$(SUDO) docker rmi mamid/notifier

testbed_net:
	# ignore all errors for idempotence
	-$(SUDO) docker network create \
		--gateway=10.101.202.254 --subnet 10.101.202.0/24 mamidnet0 >/dev/null 2>&1

docker/testbed_builder.depend: docker/builder.dockerfile
	$(SUDO) docker build -f=docker/builder.dockerfile -t=mamid/builder .
	touch docker/testbed_builder.depend

docker/.dockergopath:
	mkdir -p $@

DOCKERBUILD_GOPATH = /gopath
DOCKERBUILD_SRCDIR = $(DOCKERBUILD_GOPATH)/src/github.com/KIT-MAMID/mamid

# build using this makefile and custom suffix inside the docker container
dockerbuild: docker/.dockergopath docker/testbed_builder.depend
	@echo [ INFO ] Starting build inside docker container.
	@$(SUDO) docker run -it \
		--rm=true \
		-v=`pwd`/docker/.dockergopath:$(DOCKERBUILD_GOPATH) \
	       	-v=`pwd`:$(DOCKERBUILD_SRCDIR) \
		mamid/builder \
		/gopath/src/github.com/KIT-MAMID/mamid/docker/buildDockerBinaries.bash $(DOCKERBUILD_SRCDIR)
	# docker runs as root, hence all build products are owned by root = not good
	$(SUDO) chown `id -u`:`id -g` -R docker/.dockergopath
	$(SUDO) chown `id -u`:`id -g` build/*_docker
	@echo [ INFO ] Finished build inside docker container.

docker/testbed_images.depend: dockerbuild | \
		docker/buildDockerBinaries.bash \
		build/master_docker build/slave_docker build/notifier_docker  \
		docker/master.dockerfile docker/slave.dockerfile docker/notifier.dockerfile \
		$(shell find gui/)
	$(SUDO) docker build -f=docker/slave.dockerfile -t=mamid/slave .
	$(SUDO) docker build -f=docker/master.dockerfile -t=mamid/master .
	$(SUDO) docker build -f=docker/notifier.dockerfile -t=mamid/notifier .
	@touch docker/testbed_images.depend

TESTBED_SLAVENAME_CMD := seq -f '%02g' 1 $(TESTBED_SLAVE_COUNT)

testbed_up: testbed_down testbed_net docker/testbed_images.depend

	$(SUDO) docker run -d --net="mamidnet0" --ip="10.101.202.1" --name=master --volume=$(shell pwd)/gui:/mamid/gui mamid/master
	$(SUDO) docker run -d --net="mamidnet0" --ip="10.101.202.2" --name=notifier mamid/notifier

	for i in $(shell $(TESTBED_SLAVENAME_CMD)); do \
		$(SUDO) docker run -d --net="mamidnet0" --ip="10.101.202.1$$i" --name=slave$$i mamid/slave; \
	done

testbed_down:
	# ignore errors for idempotence
	-$(SUDO) docker rm -f builder
	-$(SUDO) docker rm -f master
	-$(SUDO) docker rm -f notifier

	-for i in $(shell $(TESTBED_SLAVENAME_CMD)); do \
		$(SUDO) docker rm -f slave$$i; \
	done

########################################################################################################################

