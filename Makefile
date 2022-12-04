 ifeq (, $(shell which go))
 $(error "go not found in $(PATH), consider installing it")
 endif

BINARIES := \
  cohab-server \
  cohabcli

PATH := $(shell go env GOPATH)/bin:$(PATH)
SHELL := env PATH=$(PATH) ${SHELL}

TARGETS := $(addprefix bin/,$(BINARIES))

.PHONY: $(TARGETS)
$(TARGETS): bin/%:
	cd $(subst bin,cmd,$@) && go build -o ../../$@

.PHONY: check
check:
	go test ./...
	golangci-lint run

.PHONY: clean
clean:
	@rm -f $(BINARIES)

.PHONY: air
air:
	@[ $$(which air) ] || (echo "air not found, 'go install' it"; exit 1)
	@air

.PHONY: image-build
image-build:
	podman build -t cohab-server .
