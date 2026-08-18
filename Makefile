BINARY := git-manager
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO_PATH ?= $(HOME)/.go
GO_BIN ?= $(HOME)/.local/bin
GO_ENV := GOPATH="$(GO_PATH)"
GO ?= go
GOFMT ?= gofmt

.PHONY: build install test vet fmt clean

build:
	$(GO_ENV) $(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install:
	mkdir -p "$(GO_BIN)"
	$(GO_ENV) GOBIN="$(GO_BIN)" $(GO) install -ldflags "$(LDFLAGS)" .

test:
	$(GO_ENV) $(GO) test ./...

vet:
	$(GO_ENV) $(GO) vet ./...

fmt:
	$(GO_ENV) $(GOFMT) -w .

clean:
	rm -rf bin
