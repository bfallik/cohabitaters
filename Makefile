 ifeq (, $(shell which go))
 $(error "go not found in $(PATH), consider installing it")
 endif

BINARIES := \
  cohab-server \
  cohabcli

PATH := $(shell go env GOPATH)/bin:$(PATH)
SHELL := env PATH=$(PATH) /bin/bash

TARGETS := $(addprefix bin/,$(BINARIES))

.PHONY: $(TARGETS)
$(TARGETS): bin/%:
	cd $(subst bin,cmd,$@) && go build -o ../../$@

.PHONY: check
check:
	go test ./...

.PHONY: clean
clean:
	@rm -f $(BINARIES)
