# AGENTS.md

Go 1.25+. Cobra CLI. Single binary; no runtime dependencies.

## Defaults

- TDD: failing tests first, table-driven, edge cases in separate functions
- Use `bd` (beads) for task tracking — not markdown todo lists
- Use `context7` MCP server for external library docs
- Act without confirmation unless blocked by missing info or irreversibility
- When stuck (cryptic errors, multiple failed approaches): escalate via Task tool with `subagent_type: "diagnose"`

## Documentation First

Before planning or writing code, read:

- `agentmap-design.md` -- full specification; nav block format; CLI commands; parser rules; keyword extraction; config
- `@.opencode/commands/orchestrate.md` -- read before starting any feature/bugfix/refactor work

## Fast Path

```bash
make build      # build binary to ./agentmap

# CI before pushing (required)
scripts/agent-run.sh make ci

# Targeted tests
scripts/agent-run.sh make test

# Lint
scripts/agent-run.sh make lint

# Format
make fmt

# Rebuild after code changes
make build
```

**Always wrap build/test/lint/long output commands with `scripts/agent-run.sh`** — captures verbose output to `.agent-output/`, shows only summary.

## Benchmarks

- Run `make bench` for a concise throughput summary.
- Run `make bench-update` to refresh `benchmarks.md` with the current baseline and raw data.
- Use `benchmarks.md` as the comparison point for future regressions.

## Debugging

When nav output looks wrong, use `generate -D <file>` to see what the parser found:

```bash
agentmap generate -D docs/authentication.md
```

This prints every heading found, its line number, computed section range, and size.
Compare against the file to spot missing/duplicate headings or incorrect boundaries.

## Project Structure

```
cmd/agentmap/       -- CLI entry point (cobra commands)
internal/
  parser/           -- markdown heading parser (h1-h3; code fence aware)
  navblock/         -- nav block parser/writer (read/write AGENT:NAV blocks)
  keywords/         -- Tier 1 keyword extraction (TF-IDF; stopwords)
  gitutil/          -- git diff integration (changed line ranges -> sections)
  config/           -- agentmap.yml loader and defaults
testdata/           -- sample markdown files for tests
```

## Conventions

- **Assertions:** stdlib only. `reflect.DeepEqual` and manual checks. No testify, no go-cmp.
- **Error wrapping:** `fmt.Errorf("context: %w", err)` -- always wrap, never swallow.
- **CLI framework:** cobra. No viper -- simple YAML config via `gopkg.in/yaml.v3`.
- **No LLM calls** in `update` or `check` commands -- ever.
- **Target performance:** `update` and `check` under 100ms for 50 files.
- **File I/O:** read each file once, write only if changed.
- **Never alter test data files** -- always use `--output /tmp/` or `--dry-run` when testing on real files

## Key Design Constraints (from agentmap-design.md)

- No commas in `about` or `why` fields -- use semicolons
- No quoting or escaping in nav blocks
- `update` never writes descriptions -- only refreshes line numbers
- `check` never modifies files -- only validates
- `generate` overwrites existing nav blocks entirely
- Maximum tracked heading depth: h3 (configurable via `max_depth`)
- Files under `min_lines` (default 50) get purpose-only blocks

## Reading Markdown Files

Every markdown file over 50 lines has an AGENT:NAV block in the first
20 lines. Read it before reading the rest of the file.

- If purpose doesn't match your task; stop reading.
- Use s,e line ranges: Read(offset=s, limit=e-s) for the section you need.
- Check see before searching; the file you need may be listed.

## Before Committing Markdown Changes

1. Run: `agentmap update <changed files>` (once the tool is built)
2. Review output for sections marked content-changed or new.
3. Read flagged sections and update their descriptions in the nav block.
4. Commit.

## Commit Messages

Conventional Commits with `Generated-by` trailer:

```
feat(parser): add code fence tracking

Generated-by: <your-model-name>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`. Scope is optional.
