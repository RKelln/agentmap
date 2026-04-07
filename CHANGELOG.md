# Changelog

All notable changes to this project will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0-rc.4] - Upgrade release-note visibility - 2026-04-07

This release candidate improves update transparency by showing where to read release notes before applying an upgrade.

### Added
- `agentmap upgrade --check` now prints the target release URL so users can review notes before updating.
- `agentmap upgrade` now also prints the target release URL before download begins.

### Changed
- Upgrade output now includes release-note context in both check-only and full-update paths.

## [v0.1.0-rc.3] - Prerelease upgrade detection fix - 2026-04-06

This release candidate fixes `agentmap upgrade` behavior for prerelease installs. RC users can now detect newer RC releases instead of seeing "no releases found for your platform".

### Fixed
- `agentmap upgrade` now enables prerelease discovery when the running binary version is itself a prerelease tag (for example `v0.1.0-rc.2`).
- `agentmap upgrade --check` now correctly reports availability for newer RC releases instead of returning a false "no releases" result.

### Infrastructure
- Added unit coverage for prerelease detection logic in `cmd/agentmap/upgrade_test.go`.

## [v0.1.0-rc.2] - Index and installer hardening - 2026-04-05

This release candidate focuses on reliability for real repos: safer index behavior, cleaner discovery defaults, and installer fixes that work before the first stable release.

### Added
- `index` discovery now uses the shared discovery pipeline and is AGENTMAP-first in generated guidance, with a clear `AGENTMAP -> AGENT:NAV -> section` flow.
- Default markdown-control excludes now cover generated or agent-instruction files (`AGENTMAP.md`, `AGENTS.md`, `CLAUDE.md`, `LICENSE.md`) in addition to `.agentmap/**`.
- Hidden-directory exclusion is now applied by default during markdown discovery.

### Fixed
- Fixed installer prerelease behavior when no stable tag exists: install now falls back from `/releases/latest` to newest prerelease.
- Fixed installer checksum matching to validate by archive filename (not temp-path), resolving `checksums.txt` lookup failures.
- Fixed installer version-resolution output so status text does not corrupt download URLs during command substitution.
- Fixed `index` safety: files containing an `AGENT:NAV` marker but unparsable blocks are no longer overwritten unless `--force` is used.
- Fixed exclude pattern handling for recursive patterns like `agents/**` in config-driven discovery.

### Changed
- `index` no longer injects the `See AGENTMAP.md for the full file index.` pointer line into `AGENTS.md` in dedicated index mode.
- Agent setup templates are now DRY via a shared body placeholder expansion path, reducing drift across tool-specific templates.
- Shared guidance was tightened to minimize token use and discourage grep-first behavior.

### Infrastructure
- CI and release workflows now use `actions/checkout@v5` and `actions/setup-go@v6` for Node 24-compatible action runtime support.

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
