.PHONY: build test lint fmt install clean check-sample

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

# Run agentmap generate on a copy of the design doc (with fake nav block stripped)
# and diff the result against the expected output. Sanity check, not a true test.
SAMPLE := /tmp/agentmap-design-sample.md
EXPECTED := testdata/agentmap-design-expected.md

check-sample: build
	@# Strip the existing fake nav block from the design doc into a temp copy
	@sed '/<!-- AGENT:NAV/,/-->/d' agentmap-design.md > $(SAMPLE)
	@# Generate a nav block on that single file, write to temp output
	@./$(BINARY) generate $(SAMPLE) -o $(SAMPLE)
	@# Diff against expected output
	@diff -u $(EXPECTED) $(SAMPLE) || { \
		echo "Sample output changed. Review the diff above."; \
		echo "If changes are expected, run: cp $(SAMPLE) $(EXPECTED)"; \
		rm -f $(SAMPLE); \
		exit 1; \
	}
	@rm -f $(SAMPLE)
	@echo "check-sample: OK"

# CI gate: all three must pass before merge
ci: test lint build
