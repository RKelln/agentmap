.PHONY: build test lint install clean

BINARY := agentmap
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/agentmap

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/agentmap

clean:
	rm -f $(BINARY)

ci: test lint build
