GO=`which go`

.PHONY: all
all: master/master slave/slave


.PHONY: clean
clean: clean_master clean_slave


master/master:
	cd master && go build master.go

.PHONY:clean_master
clean_master:
	cd master/ && go clean
	rm -f master/master

slave/slave:
	cd slave && go build slave.go controller.go

.PHONY: clean_slave
clean_slave:
	cd slave/ && go clean
	rm -f slave/slave
