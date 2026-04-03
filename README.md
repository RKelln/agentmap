# agentmap

Navigation maps for AI agents.

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

## Why

Code files have LSPs, treesitter, go-to-definition, and symbol search. Markdown has none of these. When an agent needs information from a 500-line markdown file, it typically reads the entire file (~2000 tokens) even if it only needs a 20-line section (~80 tokens).

`agentmap` closes that gap with a compact nav block that collapses multi-step navigation into a single precise `Read(offset=s, limit=n)` call.

## For AI Agents

If you are an AI coding agent: `agentmap` adds navigation maps to markdown files so you can read exactly the section you need instead of entire files.

To set up agentmap in your project:

```bash
agentmap init
```

This detects your agent tool (Claude Code; Cursor; Windsurf; Continue; Roo Code; OpenCode; GitHub Copilot; etc.) and installs the right configuration files automatically.

See [AGENTS.md](AGENTS.md) for the full workflow: how to read nav blocks and how to update them after editing markdown files.

## Install

```bash
# macOS / Linux
curl -sSfL https://raw.githubusercontent.com/RKelln/agentmap/main/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/RKelln/agentmap/main/install.ps1 | iex

# Homebrew
brew install RKelln/agentmap/agentmap

# Go
go install github.com/ryankelln/agentmap/cmd/agentmap@latest

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

Generate nav blocks for all markdown files in your docs directory:

```bash
agentmap generate ./docs
```

Refresh line numbers after editing:

```bash
agentmap update ./docs/authentication.md
```

Validate before committing:

```bash
agentmap check ./docs
```

Set up agent tool configuration:

```bash
agentmap init
```

## Commands

| Command | Purpose |
|---|---|
| `agentmap generate [path]` | Create nav blocks for markdown files |
| `agentmap update [path]` | Refresh line numbers; preserve descriptions |
| `agentmap check [path]` | Validate nav blocks are in sync |
| `agentmap init [path]` | Configure your agent tool to use agentmap |
| `agentmap uninit [path]` | Remove agentmap configuration injected by init |
| `agentmap upgrade` | Upgrade agentmap to the latest version |
| `agentmap uninstall` | Remove agentmap binary and config |
| `agentmap index [path]` | Bulk index files and generate task list |

## CI Integration

Two-line GitHub Actions integration:

```yaml
- uses: RKelln/agentmap@main
```

Or with a pinned version:

```yaml
- uses: RKelln/agentmap@v0.1.0
  with:
    path: ./docs
```

## Configuration

Optional `agentmap.yml` in your project root:

```yaml
min_lines: 50
sub_threshold: 50
expand_threshold: 150
max_depth: 3
```

## Design

See [agentmap-design.md](agentmap-design.md) for the full specification including nav block format, CLI behavior, parser rules, keyword extraction, and token budget analysis.

## Contributing

1. Fork the repo and create a feature branch
2. Write tests first (table-driven, stdlib only)
3. Run `make ci` — test + lint + build must all pass
4. Open a pull request

## License

MIT. See [LICENSE](LICENSE).
