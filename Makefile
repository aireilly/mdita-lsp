BINARY := mdita-lsp
PKG := github.com/aireilly/mdita-lsp
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint install clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/mdita-lsp

test:
	go test -race ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/mdita-lsp

clean:
	rm -f $(BINARY)
