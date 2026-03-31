# agentmap

Navigation maps for AI agents.

## What It Does

`agentmap` generates and maintains compact navigation blocks at the top of markdown files:

```markdown
<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange; refresh and revocation
nav[6]{s,e,name,about}:
12,65,#Authentication,token lifecycle management
14,35,##Token Exchange,OAuth2 code-for-token flow
36,50,##Token Refresh,silent rotation and expiry detection
see[2]{path,why}:
src/config.py,default timeout and token TTL values
docs/error-policy.md,error handling for auth failures
-->
```

This gives AI coding agents:
1. A one-line file purpose (should I read this file at all?)
2. Section index with line ranges (which section do I need, and exactly which lines?)
3. Cross-references to related files (is the answer actually somewhere else?)

## Commands

| Command | Purpose |
|---|---|
| `agentmap generate [path]` | Create nav blocks for markdown files |
| `agentmap update [path]` | Refresh line numbers; preserve descriptions |
| `agentmap check [path]` | Validate nav blocks are in sync |

## Install

```bash
go install github.com/ryankelln/agentmap/cmd/agentmap@latest
```

Or build from source:

```bash
git clone https://github.com/ryankelln/agentmap.git
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

## Configuration

Optional `agentmap.yml` in your project root:

```yaml
min_lines: 50
sub_threshold: 50
expand_threshold: 150
max_depth: 3
```

See `agentmap-design.md` for the full specification.
