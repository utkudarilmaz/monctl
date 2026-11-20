BINARY := bin/monctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/monctl

test:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin dist
