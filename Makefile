 ifeq (, $(shell which go))
 $(error "go not found in $(PATH), consider installing it")
 endif

# gh release download --pattern '*linux-x64' --repo tailwindlabs/tailwindcss
 ifeq (, $(shell which tailwindcss))
 $(error "tailwindcss not found in $(PATH), consider installing it")
 endif

BINARIES := \
  cohab-server \
  cohabcli

PATH := $(shell go env GOPATH)/bin:$(PATH)
SHELL := env PATH=$(PATH) ${SHELL}

 ifeq (, $(shell which sqlc))
 $(error "sqlc not found in $(PATH), `go install` it")
 endif

TARGETS := $(addprefix bin/,$(BINARIES))

.PHONY: $(TARGETS)
$(TARGETS): bin/%:
	cd $(subst bin,cmd,$@) && go build -o ../../$@
	@(go version -m $@ | grep -q build) || (echo "vcs info not found"; exit 1)

.PHONY: check
check:
	sqlc vet
	go test ./...
	golangci-lint run
	shellcheck deploy/*.sh

.PHONY: generate-sqlc
generate-sqlc:
	sqlc vet && sqlc generate

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

.PHONY: tailwind-css-output
tailwind-css-output:
	tailwindcss -i html/tailwindcss-src/input.css -o html/tailwindcss-dist/output.css
