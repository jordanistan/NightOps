GO ?= go
GOFLAGS ?= -buildvcs=false
export GOFLAGS
BINARY := nightops
CLI_BINARY := nightopsctl

.PHONY: all build build-cli test fmt vet verify run clean

all: verify

build:
	$(GO) build -o bin/$(BINARY) ./cmd/nightops

build-cli:
	$(GO) build -o bin/$(CLI_BINARY) ./cmd/nightopsctl

test:
	$(GO) test ./...

fmt:
	gofmt -w cmd internal

vet:
	$(GO) vet ./...

verify: fmt test vet build build-cli

run:
	$(GO) run ./cmd/nightops

clean:
	rm -rf bin
