.PHONY: build test lint fmt install clean check-sample bench bench-update smoke smoke-install

BINARY := agentmap
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo '')"
GOBIN := $(shell go env GOPATH)/bin

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/agentmap

test:
	go test ./... -v -race

lint:
	$(GOBIN)/golangci-lint run ./...

fmt:
	$(GOBIN)/golangci-lint fmt ./...

install:
	go install $(LDFLAGS) ./cmd/agentmap

clean:
	rm -f $(BINARY)

agent-clean:
	scripts/agent-cleanup.sh

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

bench:
	scripts/agent-bench.sh

bench-update:
	scripts/agent-bench.sh --write benchmarks.md

# Smoke tests: run the compiled binary against real testdata.
# Catches ldflags injection, embedded asset loading, and CLI issues
# that unit tests (which use the package API) cannot catch.
# Pass BINARY=<path> to test a specific build (e.g. a goreleaser snapshot).
smoke: build
	scripts/smoke.sh ./$(BINARY)

# Smoke test the install.sh script inside a clean Ubuntu container.
# Requires Docker. Tests the end-to-end install path for new users.
smoke-install:
	docker run --rm ubuntu:22.04 bash -c "\
		apt-get update -qq && apt-get install -q -y curl && \
		curl -fsSL https://raw.githubusercontent.com/RKelln/agentmap/main/install.sh | sh && \
		agentmap version"
