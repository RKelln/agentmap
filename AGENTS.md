# AGENTS.md

## Documentation First

Before planning or writing code, read:

- `agentmap-design.md` -- full specification; nav block format; CLI commands; parser rules; keyword extraction; config
- `@.opencode/commands/orchestrate.md` -- read before starting any feature/bugfix/refactor work

## Project Overview

**agentmap** is a CLI tool that generates and maintains compact navigation blocks (`<!-- AGENT:NAV -->`) at the top of markdown files. Written in Go. Single binary; no runtime dependencies.

## Build and Test

```bash
make build      # build binary to ./agentmap
make test       # run all tests
make lint       # run golangci-lint
make install    # install to $GOPATH/bin
```

CI gate: `make test && make lint && make build` -- all three must pass before merge.

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

- **Test style:** table-driven tests; edge cases in separate test functions
- **CLI framework:** cobra
- **Config format:** YAML (`agentmap.yml`); loaded with `gopkg.in/yaml.v3`
- **No LLM calls** in `update` or `check` commands -- ever
- **Target performance:** `update` and `check` under 100ms for 50 files
- **Commit messages:** Conventional Commits with `Generated-by` trailer:
  ```
  feat(parser): add code fence tracking

  Generated-by: <your-model-name>
  ```
  Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`. Scope is optional.

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

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
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
