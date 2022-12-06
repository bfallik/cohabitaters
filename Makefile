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
	shellcheck deploy/*.sh

.PHONY: clean
clean:
	rm -f $(TARGETS)

.PHONY: air
air:
	@[ $$(which air) ] || (echo "air not found, 'go install' it"; exit 1)
	@air

.PHONY: image-build
image-build:
	podman build -t cohab-server .

.PHONY: image-fedora-test-build
image-fedora-test-build:
	podman build -f Dockerfile.fedora -t fedora-test-build .
	podman rmi fedora-test-build
