.PHONY: build test lint fmt install clean

BINARY := agentmap
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
GOBIN := $(shell go env GOPATH)/bin

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/agentmap

test:
	go test ./... -v -race

lint:
	$(GOBIN)/golangci-lint run ./...

fmt:
	$(GOBIN)/gofumpt -w .

install:
	go install $(LDFLAGS) ./cmd/agentmap

clean:
	rm -f $(BINARY)

# CI gate: all three must pass before merge
ci: test lint build
