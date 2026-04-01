# Benchmarks

Baseline captured on 2026-04-01 on `x86_64` / `Linux 6.17.0-19-generic`.

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
| small synthetic | ~18 lines | 0.024 | 185 |
| medium synthetic | ~82 lines | 0.076 | 636 |
| large synthetic | ~216 lines | 0.191 | 1683 |
| design-clean | 850 lines | 4.621 | 23235 |

### Keyword helpers

These are text-level helpers, not full files, so the numbers are shown separately.

| Case | ms/op | Allocations |
|---|---:|---:|
| ExtractKeywords / small | 0.003 | 36 |
| ExtractKeywords / medium | 0.020 | 207 |
| ExtractKeywords / large | 0.077 | 785 |
| ExtractPurpose | 0.045 | 464 |

### Raw benchmark data

| Package | Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|---:|
| generate | small | 24278 | 13617 | 185 |
| generate | medium | 75568 | 51662 | 636 |
| generate | large | 191380 | 113411 | 1683 |
| generate | design-clean | 4620777 | 2292661 | 23235 |
| keywords | ExtractKeywords/small | 3295 | 1552 | 36 |
| keywords | ExtractKeywords/medium | 19644 | 11216 | 207 |
| keywords | ExtractKeywords/large | 76679 | 41680 | 785 |
| keywords | ExtractPurpose | 44635 | 23280 | 464 |

## Notes

- `generate` includes the full dry-run file path, so it is the main throughput baseline.
- `design-clean` is the real long-document fixture and is the best regression comparison for large docs.
- Keep future benchmark entries append-only when possible so history stays visible.
