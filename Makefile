 ifeq (, $(shell which go))
 $(error "go not found in $(PATH), consider installing it")
 endif

PATH := $(shell go env GOPATH)/bin:$(PATH)
SHELL := env PATH=$(PATH) /bin/bash

bin/cohabcli:
	cd $(subst bin,cmd,$@) && go build -o ../../$@

.PHONY: check
check:
	go test ./...

.PHONY: clean
clean:
	@rm -f bin/cohabcli
