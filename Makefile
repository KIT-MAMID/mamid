GO ?= $(shell which go)
GOFMT ?= $(shell which gofmt)
GREP ?= $(shell which grep)
GOOS ?= $(shell uname | tr A-Z a-z)
GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILD_SUFFIX = $(GOOS)_$(GOARCH)
TESTBED_SLAVE_COUNT ?= 3
pkgs          = $(shell $(GO) list ./... | grep -v /vendor/)
pkg_dirs      = $(addprefix $(GOPATH)/src/,$(pkgs))

.PHONY: all
all: build/master_$(BUILD_SUFFIX) build/slave_$(BUILD_SUFFIX)


.PHONY: clean
clean: clean_master clean_slave clean_testbed


build/master_$(BUILD_SUFFIX):
	cd master/cmd && $(GO) build -o ../../build/master_$(BUILD_SUFFIX)

.PHONY:clean_master
clean_master:
	cd master/ && $(GO) clean
	rm -rf build/master*


build/slave_$(BUILD_SUFFIX):
	cd slave/cmd && $(GO) build -o ../../build/slave_$(BUILD_SUFFIX)

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


.PHONY: testbed_up testbed_down testbed_net


clean_testbed: testbed_down
	rm -f docker/*.depend
	-sudo docker rmi mamid/master
	-sudo docker rmi mamid/slave

testbed_net:
	# ignore all errors for idempotence
	-sudo docker network create \
		--gateway=10.101.202.254 --subnet 10.101.202.0/24 mamidnet0 >/dev/null 2>&1

docker/testbed_images.depend: build/master_linux_amd64 build/slave_linux_amd64 docker/slave.dockerfile docker/master.dockerfile
	sudo docker build -f=docker/slave.dockerfile -t=mamid/slave .
	sudo docker build -f=docker/master.dockerfile -t=mamid/master .
	touch docker/testbed_images.depend

TESTBED_SLAVENAME_CMD := seq -f '%02g' 1 $(TESTBED_SLAVE_COUNT)

testbed_up: testbed_down testbed_net docker/testbed_images.depend

	sudo docker run -d --net="mamidnet0" --ip="10.101.202.1" --name=master mamid/master

	for i in $(shell $(TESTBED_SLAVENAME_CMD)); do \
		sudo docker run -d --net="mamidnet0" --ip="10.101.202.1$$i" --name=slave$$i mamid/slave; \
	done

testbed_down:
	# ignore errors for idempotence
	-sudo docker rm -f master

	-for i in $(shell $(TESTBED_SLAVENAME_CMD)); do \
		sudo docker rm -f slave$$i; \
	done

