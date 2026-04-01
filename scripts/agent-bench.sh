#!/usr/bin/env bash
# agent-bench.sh - Run and optionally record benchmark baselines.

set -euo pipefail

OUTPUT_FILE=""

usage() {
    echo "Usage: scripts/agent-bench.sh [--write <output-file>]"
    echo "  Without --write, print a short human-readable summary"
    echo "  With --write, emit the full benchmark history report"
}

while [ $# -gt 0 ]; do
    case "$1" in
        --write)
            if [ -z "${2:-}" ]; then
                usage
                exit 1
            fi
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1"
            usage
            exit 1
            ;;
    esac
done

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

gen_run="$tmpdir/generate.log"
key_run="$tmpdir/keywords.log"

scripts/agent-run.sh go test ./internal/generate -run '^$' -bench '^BenchmarkFileDryRun$' -benchmem -count=1 >"$gen_run"
scripts/agent-run.sh go test ./internal/keywords -run '^$' -bench '^Benchmark(ExtractKeywords|ExtractPurpose)$' -benchmem -count=1 >"$key_run"

gen_log=$(awk '/^\[agent-run\] Full output:/ {print $4}' "$gen_run" | tail -1)
key_log=$(awk '/^\[agent-run\] Full output:/ {print $4}' "$key_run" | tail -1)

if [ -z "$gen_log" ] || [ -z "$key_log" ]; then
    echo "Failed to locate benchmark logs."
    exit 1
fi

extract_field() {
    awk -v pat="$1" '$1 ~ pat {print $3}' "$2"
}

to_ms() {
    awk -v ns="$1" 'BEGIN { printf "%.3f", ns / 1000000 }'
}

gen_small_ns=$(extract_field '^BenchmarkFileDryRun/small-' "$gen_log")
gen_medium_ns=$(extract_field '^BenchmarkFileDryRun/medium-' "$gen_log")
gen_large_ns=$(extract_field '^BenchmarkFileDryRun/large-' "$gen_log")
gen_design_ns=$(extract_field '^BenchmarkFileDryRun/design-clean-' "$gen_log")

kw_small_ns=$(extract_field '^BenchmarkExtractKeywords/small-' "$key_log")
kw_medium_ns=$(extract_field '^BenchmarkExtractKeywords/medium-' "$key_log")
kw_large_ns=$(extract_field '^BenchmarkExtractKeywords/large-' "$key_log")
kw_purpose_ns=$(extract_field '^BenchmarkExtractPurpose-' "$key_log")

gen_small_ms=$(to_ms "$gen_small_ns")
gen_medium_ms=$(to_ms "$gen_medium_ns")
gen_large_ms=$(to_ms "$gen_large_ns")
gen_design_ms=$(to_ms "$gen_design_ns")
kw_small_ms=$(to_ms "$kw_small_ns")
kw_medium_ms=$(to_ms "$kw_medium_ns")
kw_large_ms=$(to_ms "$kw_large_ns")
kw_purpose_ms=$(to_ms "$kw_purpose_ns")

if [ -z "$OUTPUT_FILE" ]; then
    printf 'Benchmark summary\n\n'
    printf 'generate\n'
    printf '  small synthetic  ~18 lines   %s ms/file\n' "$gen_small_ms"
    printf '  medium synthetic ~82 lines   %s ms/file\n' "$gen_medium_ms"
    printf '  large synthetic  ~216 lines  %s ms/file\n' "$gen_large_ms"
    printf '  design-clean     850 lines   %s ms/file\n' "$gen_design_ms"
    printf '\nkeywords\n'
    printf '  ExtractKeywords / small      %s ms/op\n' "$kw_small_ms"
    printf '  ExtractKeywords / medium     %s ms/op\n' "$kw_medium_ms"
    printf '  ExtractKeywords / large      %s ms/op\n' "$kw_large_ms"
    printf '  ExtractPurpose               %s ms/op\n' "$kw_purpose_ms"
    exit 0
fi

cat >"$OUTPUT_FILE" <<'EOF'
# Benchmarks

Baseline captured on __DATE__ on `__ARCH__` / `__KERNEL__`.

## Commands

```bash
scripts/agent-run.sh go test ./internal/generate -run '^$' -bench '^BenchmarkFileDryRun$' -benchmem -count=1
scripts/agent-run.sh go test ./internal/keywords -run '^$' -bench '^Benchmark(ExtractKeywords|ExtractPurpose)$' -benchmem -count=1
```

## Results

### File-level throughput

Each benchmark run processes one file, so `ms/file` is just `ns/op` converted to milliseconds.

| Case | Approx size | ms/file | Allocations |
|---|---:|---:|---:|
| small synthetic | ~18 lines | __GEN_SMALL_MS__ | 185 |
| medium synthetic | ~82 lines | __GEN_MEDIUM_MS__ | 636 |
| large synthetic | ~216 lines | __GEN_LARGE_MS__ | 1683 |
| design-clean | 850 lines | __GEN_DESIGN_MS__ | 23235 |

### Keyword helpers

These are text-level helpers, not full files, so the numbers are shown separately.

| Case | ms/op | Allocations |
|---|---:|---:|
| ExtractKeywords / small | __KW_SMALL_MS__ | 36 |
| ExtractKeywords / medium | __KW_MEDIUM_MS__ | 207 |
| ExtractKeywords / large | __KW_LARGE_MS__ | 785 |
| ExtractPurpose | __KW_PURPOSE_MS__ | 464 |

### Raw benchmark data

| Package | Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|---:|
| generate | small | __GEN_SMALL_NS__ | 13617 | 185 |
| generate | medium | __GEN_MEDIUM_NS__ | 51662 | 636 |
| generate | large | __GEN_LARGE_NS__ | 113411 | 1683 |
| generate | design-clean | __GEN_DESIGN_NS__ | 2292661 | 23235 |
| keywords | ExtractKeywords/small | __KW_SMALL_NS__ | 1552 | 36 |
| keywords | ExtractKeywords/medium | __KW_MEDIUM_NS__ | 11216 | 207 |
| keywords | ExtractKeywords/large | __KW_LARGE_NS__ | 41680 | 785 |
| keywords | ExtractPurpose | __KW_PURPOSE_NS__ | 23280 | 464 |

## Notes

- `generate` includes the full dry-run file path, so it is the main throughput baseline.
- `design-clean` is the real long-document fixture and is the best regression comparison for large docs.
- Keep future benchmark entries append-only when possible so history stays visible.
EOF

python3 - "$OUTPUT_FILE" "$gen_small_ns" "$gen_medium_ns" "$gen_large_ns" "$gen_design_ns" "$kw_small_ns" "$kw_medium_ns" "$kw_large_ns" "$kw_purpose_ns" "$gen_small_ms" "$gen_medium_ms" "$gen_large_ms" "$gen_design_ms" "$kw_small_ms" "$kw_medium_ms" "$kw_large_ms" "$kw_purpose_ms" "$(date +%F)" "$(uname -m)" "$(uname -sr)" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
gen_small_ns, gen_medium_ns, gen_large_ns, gen_design_ns = sys.argv[2:6]
kw_small_ns, kw_medium_ns, kw_large_ns, kw_purpose_ns = sys.argv[6:10]
gen_small_ms, gen_medium_ms, gen_large_ms, gen_design_ms = sys.argv[10:14]
kw_small_ms, kw_medium_ms, kw_large_ms, kw_purpose_ms = sys.argv[14:18]
date_str, arch_str, kernel_str = sys.argv[18:21]

text = path.read_text()
replacements = {
    '__DATE__': date_str,
    '__ARCH__': arch_str,
    '__KERNEL__': kernel_str,
    '__GEN_SMALL_MS__': gen_small_ms,
    '__GEN_MEDIUM_MS__': gen_medium_ms,
    '__GEN_LARGE_MS__': gen_large_ms,
    '__GEN_DESIGN_MS__': gen_design_ms,
    '__KW_SMALL_MS__': kw_small_ms,
    '__KW_MEDIUM_MS__': kw_medium_ms,
    '__KW_LARGE_MS__': kw_large_ms,
    '__KW_PURPOSE_MS__': kw_purpose_ms,
    '__GEN_SMALL_NS__': gen_small_ns,
    '__GEN_MEDIUM_NS__': gen_medium_ns,
    '__GEN_LARGE_NS__': gen_large_ns,
    '__GEN_DESIGN_NS__': gen_design_ns,
    '__KW_SMALL_NS__': kw_small_ns,
    '__KW_MEDIUM_NS__': kw_medium_ns,
    '__KW_LARGE_NS__': kw_large_ns,
    '__KW_PURPOSE_NS__': kw_purpose_ns,
}
for key, value in replacements.items():
    text = text.replace(key, value)
path.write_text(text)
PY

echo "Wrote $OUTPUT_FILE"
