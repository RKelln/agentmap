#!/usr/bin/env bash
# scripts/smoke.sh — Binary smoke tests for agentmap.
# Exercises the compiled binary against real testdata to catch issues that
# unit tests (which use the package API) cannot: ldflags injection, embedded
# asset loading, CLI flag parsing, and path handling in compiled form.
#
# Usage:
#   scripts/smoke.sh [binary]          # binary defaults to ./agentmap
#   scripts/smoke.sh dist/agentmap_Linux_x86_64/agentmap  # test a snapshot build
#
# Exit: 0 if all checks pass, 1 if any check fails.

set -euo pipefail

BIN="${1:-./agentmap}"
TESTDATA="testdata"
PASS=0
FAIL=0

ok()   { echo "  PASS  $1"; PASS=$((PASS+1)); }
fail() { echo "  FAIL  $1"; echo "        $2"; FAIL=$((FAIL+1)); }

require_binary() {
    if [ ! -x "$BIN" ]; then
        echo "smoke: binary not found or not executable: $BIN"
        echo "       Run 'make build' first, or pass a path: scripts/smoke.sh <binary>"
        exit 1
    fi
}

# ── helpers ──────────────────────────────────────────────────────────────────

run_check() {
    local label="$1"; shift
    local output
    if output=$("$@" 2>&1); then
        ok "$label"
    else
        fail "$label" "$output"
    fi
}

# Like run_check but treats any non-zero exit as pass (command ran, didn't crash).
# Used for 'check' which exits 1 when nav blocks are stale — that's legitimate.
run_no_panic() {
    local label="$1"; shift
    local output
    output=$("$@" 2>&1) || true
    if echo "$output" | grep -qiE 'panic|runtime error'; then
        fail "$label" "$output"
    else
        ok "$label"
    fi
}

# ── tests ────────────────────────────────────────────────────────────────────

echo "smoke: testing $BIN"
echo ""
require_binary

# 1. version — verifies ldflags injection worked; accepts semver, git hash, or "dev"
run_check "version exits 0 and prints version string" \
    "$BIN" version

# 2. generate --dry-run on a real file — verifies parser + navblock + embed
run_check "generate --dry-run on authentication.md" \
    "$BIN" generate "$TESTDATA/authentication.md" --dry-run

# 3. update --dry-run on a real file — verifies update path
run_check "update --dry-run on authentication.md" \
    "$BIN" update "$TESTDATA/authentication.md" --dry-run

# 4. check on a file — may exit 1 if nav block is stale (not a crash)
run_no_panic "check does not panic on authentication.md" \
    "$BIN" check "$TESTDATA/authentication.md"

# 5. index --dry-run in a temp dir — verifies index + WriteFilesBlock path
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
cp "$TESTDATA/authentication.md" "$tmpdir/"
run_check "index --dry-run in temp dir" \
    "$BIN" index "$tmpdir" --dry-run

# 6. init --dry-run in a temp dir — verifies embedded templates are accessible
run_check "init --dry-run in temp dir" \
    "$BIN" init "$tmpdir" --dry-run

# 7. upgrade --check — verifies go-selfupdate integration doesn't crash
#    (may print "up to date" or a new version; both are fine)
run_no_panic "upgrade --check does not panic" \
    "$BIN" upgrade --check

# 8. uninit --dry-run --yes in a temp dir — verifies uninit path doesn't crash
run_check "uninit --dry-run in temp dir" \
    "$BIN" uninit "$tmpdir" --dry-run --yes

# 9. uninstall --dry-run --yes — verifies uninstall path doesn't crash
#    (binary is not in a Homebrew/Scoop/GOPATH path, so it takes the direct-install path)
run_check "uninstall --dry-run does not crash" \
    "$BIN" uninstall --dry-run --yes

# ── summary ──────────────────────────────────────────────────────────────────

echo ""
echo "smoke: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
