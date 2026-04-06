# Changelog

All notable changes to this project will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0-rc.1] - Introducing agentmap - 2026-04-05

`agentmap` is a local CLI that makes markdown files navigable for coding agents.
It writes compact `AGENT:NAV` blocks so agents can decide quickly whether a file
is relevant, jump to exact sections, and follow cross-file references without
reading entire long documents.

### What agentmap does

- Adds a one-line `purpose` for fast file-level triage.
- Adds section entries with `s,n` ranges so agents can call `Read(offset=s, limit=n)` directly.
- Adds optional `see` references to related files when context lives elsewhere.
- Adds `lines:N` metadata so agents know file size before reading.

### How the workflow works

1. Run `agentmap index <path>` for first-pass indexing across docs (or `agentmap generate <path>` for targeted generation).
2. Agent or human refines generated `purpose`, `about`, and `see` descriptions.
3. After markdown edits, run `agentmap update <changed files>` to refresh line numbers while preserving descriptions.
4. Run `agentmap check <path>` in CI or pre-commit to keep nav blocks in sync.

For `generate`, `index`, `update`, and `check`, the workflow is local,
deterministic, and fast (no external LLM calls required by those commands).

### What ships in this release candidate

- Core command set: `generate`, `index`, `update`, `check`, `init`, `uninit`, `uninstall`, `upgrade`, `hook`, and `version`.
- Agent setup via `agentmap init` for AGENTS-compatible workflows plus Cursor, Windsurf, Continue, Roo Code, Amazon Q, and OpenCode.
- Hook integration for `.pre-commit-config.yaml`, Husky, and plain git hooks.
- Install options for Homebrew, Scoop, shell/PowerShell scripts, and `go install`.
- Automated GitHub Actions + GoReleaser release pipeline with checksums and package repo publishing.

### Release candidate focus

`v0.1.0-rc.1` is for validating the full end-to-end experience: install,
initialize agent guidance, generate/index nav blocks, keep them updated,
and enforce checks in local workflows and CI.
