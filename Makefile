BINARY := bin/monctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
export MONCTL_VERSION := $(VERSION)

.PHONY: build test lint clean

build:
	go build -buildvcs=false -ldflags "-X main.version=$$MONCTL_VERSION" -o $(BINARY) ./cmd/monctl

test:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin dist
