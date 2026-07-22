GO ?= go
GOFLAGS ?= -buildvcs=false
export GOFLAGS
BINARY := nightops

.PHONY: all build test fmt vet verify run clean

all: verify

build:
	$(GO) build -o bin/$(BINARY) ./cmd/nightops

test:
	$(GO) test ./...

fmt:
	gofmt -w cmd internal

vet:
	$(GO) vet ./...

verify: fmt test vet build

run:
	$(GO) run ./cmd/nightops

clean:
	rm -rf bin
