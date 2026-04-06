# AGENTS.md

Go 1.25+. Cobra CLI. Single binary; no runtime dependencies.

## Defaults

- TDD: failing tests first, table-driven, edge cases in separate functions
- Use `bd` (beads) for task tracking — not markdown todo lists
- Use `context7` MCP server for external library docs
- Act without confirmation unless blocked by missing info or irreversibility
- When stuck (cryptic errors, multiple failed approaches): escalate via Task tool with `subagent_type: "diagnose"`

## Documentation First

When you need to reference the design:

- `agentmap-design.md` -- full specification; nav block format; CLI commands; parser rules; keyword extraction; config

## Fast Path

```bash
scripts/agent-run.sh make build      # build binary to ./agentmap

# CI before pushing (required)
scripts/agent-run.sh make ci

# Targeted tests
scripts/agent-run.sh make test

# Lint
scripts/agent-run.sh make lint

# Format
scripts/agent-run.sh make fmt

# Rebuild after code changes
scripts/agent-run.sh make build
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
  templates/        -- embedded agent skill templates (Go embed; written by init)
testdata/           -- sample markdown files for tests
```

## Conventions

- **Assertions:** stdlib only. `reflect.DeepEqual` and manual checks. No testify, no go-cmp.
- **Error wrapping:** `fmt.Errorf("context: %w", err)` -- always wrap, never swallow.
- **CLI framework:** cobra. No viper -- simple YAML config via `gopkg.in/yaml.v3`.
- **No external LLM calls** in `generate`; `index`; `update`; or `check` -- ever.
- These commands are local, deterministic, and designed to run quickly.
- **Target performance:** `update` and `check` under 100ms for 50 files.
- **File I/O:** read each file once, write only if changed.
- **Never alter test data files** -- always use `--output /tmp/` or `--dry-run` when testing on real files

## Key Design Constraints (from agentmap-design.md)

- No commas in nav block field values (`about`; `why`; `purpose`; subsection hints) -- use semicolons as separators
- No quoting or escaping in nav blocks
- `update` never writes descriptions -- only refreshes line numbers
- `check` never modifies files -- only validates
- `generate` overwrites existing nav blocks entirely
- Maximum tracked heading depth: h3 (configurable via `max_depth`)
- Files under `min_lines` (default 50) get purpose-only blocks

## Reading Markdown Files

Read the AGENT:NAV block at the top after any frontmatter.
Read the first 40 lines of a file before reading the rest of the file.

- If purpose doesn't match your task; stop reading.
- Use s,n line ranges: Read(offset=s; limit=n) for the section you need.
- Check see before searching; the file you need may be listed.

## Before Committing Markdown Changes

1. For first-pass indexing, run: `agentmap index <path>` (or `agentmap generate <path>` for targeted generation).
2. Review and refine nav descriptions (`purpose`; `about`; `see`) with agent/human judgment.
3. Run: `agentmap update <changed markdown files>` to refresh line numbers only.
4. Run: `agentmap check <path>` before committing.

## Commit Messages

Conventional Commits with `Generated-by` trailer:

```
feat(parser): add code fence tracking

Generated-by: <your-model-name>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`. Scope is optional.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for task tracking between issues and sessions
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Create/update beads for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
