# agentmap

Navigation maps for markdown files for AI agents.

## What It Does

`agentmap` generates and maintains compact navigation blocks at the top of markdown files:

```markdown
<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange; refresh and revocation
nav[6]{s,n,name,about}:
12,54,#Authentication,token lifecycle management
14,22,##Token Exchange,OAuth2 code-for-token flow
36,15,##Token Refresh,silent rotation and expiry detection
see[2]{path,why}:
src/config.py,default timeout and token TTL values
docs/error-policy.md,error handling for auth failures
-->
```

This gives AI coding agents:
1. A one-line file purpose (should I read this file at all?)
2. Section index with line ranges (which section do I need, and exactly which lines?)
3. Cross-references to related files (is the answer actually somewhere else?)

`agentmap` does not call external LLMs for `generate`, `update`, `check`, or
`index`. Nav block creation is local, deterministic, and fast.

## Why

Code files have LSPs, treesitter, go-to-definition, and symbol search. Markdown has none of these. When an agent needs information from a 500-line markdown file, it has to grep or read the entire file (~2000 tokens) even if it only needs a 20-line section (~80 tokens).

`agentmap` closes that gap with a compact nav block that collapses multi-step navigation into a single precise `Read(offset=s, limit=n)` call.

## For AI Agents

If you are an AI coding agent: `agentmap` adds navigation maps to markdown files so you can read exactly the section you need instead of entire files.

To set up agentmap in your project:

```bash
agentmap init
```

This detects your agent tool files and installs the right configuration automatically.

- Native tool rule files are written for Cursor, Windsurf, Continue, Roo Code, Amazon Q, and OpenCode.
- `AGENTS.md` / `CLAUDE.md` are updated for AGENTS-compatible tools (including Claude Code; GitHub Copilot coding agent workflows; and other tools that read `AGENTS.md`).
- Hooks are added when supported hook infrastructure is detected (`.pre-commit-config.yaml`; `.husky/`; or plain `.git/hooks`).

See [AGENTS.md](AGENTS.md) for the full workflow: how to read nav blocks and how to update them after editing markdown files.

## Basic Workflow

1. Run `agentmap index .` from repo root to generate nav blocks and create `.agentmap/index-tasks.md`.
2. Run `agentmap next` — it prints a self-contained prompt for the next unreviewed file, then auto-updates and checks off completed files on each subsequent call. Repeat until the task list is done.
3. After any markdown edits, run `agentmap update <changed files>` to refresh line numbers.
4. Run `agentmap check <path>` in CI/pre-commit to validate nav blocks are in sync.

`agentmap next` handles the update + check-off loop automatically — the agent just edits a file and calls `next` again. Run `agentmap guide` for instructions on writing good `purpose`, `about`, and `see` descriptions.

`index` writes outputs relative to the path you pass. Example: `agentmap index docs/` writes `docs/.agentmap/index-tasks.md` and `docs/AGENTMAP.md`.

## Install

```bash
# macOS / Linux
curl -sSfL https://raw.githubusercontent.com/RKelln/agentmap/main/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/RKelln/agentmap/main/install.ps1 | iex

# Homebrew
brew install RKelln/agentmap/agentmap

# Scoop (Windows)
scoop bucket add agentmap https://github.com/RKelln/scoop-agentmap
scoop install agentmap

# Go
go install github.com/RKelln/agentmap/cmd/agentmap@latest

# Upgrade
agentmap upgrade
```

Build from source:

```bash
git clone https://github.com/RKelln/agentmap.git
cd agentmap
make install
```

## Quick Start

First-time setup (recommended):

```bash
agentmap index .
```

This generates nav blocks for all markdown files and writes `.agentmap/index-tasks.md` — a checklist of files with auto-generated `~`-prefixed descriptions that need human or agent review.

Have an agent review and rewrite descriptions:

```bash
agentmap next   # prints a prompt for the next unreviewed file
                # on each subsequent call: auto-updates and checks off the previous file
                # repeat until the task list is empty
```

Refresh line numbers after editing markdown:

```bash
agentmap update ./docs/authentication.md
```

Validate before committing:

```bash
agentmap check ./docs
```

Optional: generate nav blocks directly without the index task list:

```bash
agentmap generate ./docs
```

Set up agent tool configuration:

```bash
agentmap init
```

## Agent + Hook Support

`agentmap init` currently configures these targets:

| Environment | What `init` updates |
|---|---|
| AGENTS-compatible tools | `AGENTS.md` (append or create fallback when nothing else is detected) |
| Claude Code | `CLAUDE.md` (if present) |
| Cursor | `.cursor/rules/agentmap.md` |
| Cursor legacy | `.cursorrules` |
| Windsurf | `.windsurf/rules/agentmap.md` |
| Continue | `.continue/rules/agentmap.md` |
| Roo Code | `.roo/rules/agentmap.md` |
| Amazon Q | `.amazonq/rules/agentmap.md` |
| OpenCode | `.opencode/skills/agentmap/SKILL.md` |
| Aider | Warning only (`.aider.conf.yml` must be updated manually with `read: [AGENTS.md]`) |

Hook handling in `init`:

| Hook system detected | `init` behavior |
|---|---|
| `.pre-commit-config.yaml` | Appends an `agentmap check .` hook |
| `.husky/` | Appends guard block to `.husky/pre-commit` |
| `.git/hooks/` (no pre-commit/husky) | Creates or appends `.git/hooks/pre-commit` |
| `.lefthook.yml` | Warns only (manual setup required) |

## Commands

| Command | Purpose |
|---|---|
| `agentmap generate [path]` | Create nav blocks for markdown files |
| `agentmap update [path]` | Refresh line numbers; preserve descriptions |
| `agentmap check [path]` | Validate nav blocks are in sync |
| `agentmap index [path]` | Bulk index files and generate task list |
| `agentmap next` | Auto-update previous file, check it off, print prompt for the next |
| `agentmap search <query> [path]` | Fuzzy search headings to surface associated content across agentmapped files |
| `agentmap headings [path]` | Print all agentmapped files with purposes and nav entries |
| `agentmap guide` | Print the nav writing guide |
| `agentmap init [path]` | Configure your agent tool to use agentmap |
| `agentmap uninit [path]` | Remove agentmap configuration injected by init |
| `agentmap hook` | Print pre-commit hook templates |
| `agentmap upgrade` | Upgrade agentmap to the latest version |
| `agentmap uninstall` | Remove agentmap binary and config |

## CI Integration

Two-line GitHub Actions integration:

```yaml
- uses: RKelln/agentmap@v0.1.0
```

Or with a specific path:

```yaml
- uses: RKelln/agentmap@v0.1.0
  with:
    path: ./docs
```

## Configuration

Optional `agentmap.yml` in your project root:

```yaml
min_lines: 50          # files under this get purpose-only blocks (default: 50)
sub_threshold: 50      # sections under this get no subsection hints (default: 50)
expand_threshold: 150  # sections over this get full h3 entries (default: 150)
max_depth: 6           # maximum heading depth to track (default: 6)
max_nav_entries: 20    # maximum nav entries per file (default: 20)
```

## Design

See [agentmap-design.md](agentmap-design.md) for the full specification including nav block format, CLI behavior, parser rules, keyword extraction, and token budget analysis.

## Contributing

1. Fork the repo and create a feature branch
2. Write tests first (table-driven, stdlib only)
3. Run `make ci` — test + lint + build must all pass
4. Open a pull request

GoReleaser is only needed locally if you're modifying `.goreleaser.yaml`. Install via `brew install goreleaser` or see [goreleaser.com](https://goreleaser.com). It is not required for normal development.

## License

MIT. See [LICENSE](LICENSE).
