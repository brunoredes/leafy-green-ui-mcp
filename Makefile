BIN     := bin/lg-mcp
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test lint fmt index inspect

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/lg-mcp

test:
	go test ./... -race

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w -local github.com/brunodonatelli/lg-mcp .

index: build
	$(BIN) index --source repo --repo-path testdata/repo --design-cache testdata/design

inspect: build
	npx @modelcontextprotocol/inspector $(BIN) --source repo --repo-path testdata/repo --design-cache testdata/design
