BINARY := mdita-lsp
PKG := github.com/aireilly/mdita-lsp
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build test lint vet fmt-check install clean publish

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/mdita-lsp

test:
	go test -race ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)

publish:
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "Building $$os/$$arch..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build $(LDFLAGS) \
			-o dist/$(BINARY)-$$os-$$arch$$ext \
			./cmd/mdita-lsp; \
	done
	@echo "Binaries in dist/"

clean:
	rm -f $(BINARY)
	rm -rf dist/
