GO=`which go`

.PHONY: all
all: master/master slave/slave


.PHONY: clean
clean: clean_master clean_slave


master/master:
	cd master && $(GO) build master.go

.PHONY:clean_master
clean_master:
	cd master/ && $(GO) clean
	rm -f master/master

slave/slave:
	cd slave && $(GO) build slave.go controller.go

.PHONY: clean_slave
clean_slave:
	cd slave/ && $(GO) clean
	rm -f slave/slave
