# Changelog

All notable changes to this project will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0-rc.8] — Nav quality and YAML frontmatter fix — 2026-04-10

Bug fix and nav description quality improvements for the `agentmap next` workflow.

### Fixed
- `agentmap update` on files with YAML frontmatter (e.g. Marp presentations) was
  merging the frontmatter closing `---` with the nav block opener `<!-- AGENT:NAV`
  into a single line (`---<!-- AGENT:NAV`). The parser could not find the nav block,
  causing `agentmap next` to loop on the same file indefinitely. A regression test
  covers this case.

### Documentation
- Guide, `agentmap next` prompt, and index task preamble now explicitly permit
  leaving `about` blank (trailing comma: `43,9,#Heading,`) when a heading is
  self-explanatory and no new information can be added. Prevents small models from
  generating noise like "Cross-section issues notes" to satisfy the rewrite
  requirement.
- Design doc (`agentmap-design.md`) updated with §11.3 rules for consecutive
  same-depth heading cluster collapse and structural (lone subtitle) tagging;
  §11.4 pruning algorithm updated to prune structural entries before depth-based
  pruning; `--drop-subtitles` flag specced for `agentmap generate` (implementation
  deferred post-v0.1.0).

## [v0.1.0-rc.7] — agentmap next: stateful single-file agent workflow — 2026-04-09

This release introduces `agentmap next`, a new command that solves context-window
overload for small models working through large index task lists. Instead of handing
a model the full 1000-line task list, `next` emits one self-contained file prompt at
a time, tracks state between calls, and automatically updates and checks off completed
files. The full agent loop is now just: edit a file, run `agentmap next`, repeat.

### Added

- **`agentmap next`** — finds the next unchecked entry in `.agentmap/index-tasks.md`
  and prints a self-contained single-file prompt (file path + current nav block +
  rewrite instructions). Solves context-window overload for large projects (e.g. 84-file
  task lists that previously required 1000+ lines of context before any work started).
- **`agentmap next --count N`** — emits N consecutive prompts (separated by `---`) in
  one call, for batched agent workflows.
- **Stateful next-state** — `agentmap next` now tracks in-flight files in
  `.agentmap/next-state` (one relPath per line). On each call it flushes state —
  running `update` + check-off on every previously-emitted file — then writes the new
  batch. The complete agent loop is: `<edit file> → agentmap next → <edit file> → agentmap next → …`
- **Blocked state** — if a file in state still has `~` descriptions when `next` is
  called, it prints a clear warning and stops, preventing silent forward progress on
  incomplete work.
- **Auto check-off on `agentmap update`** — after a successful non-dry-run update,
  `update` automatically flips `- [ ]` to `- [x]` in the task list for any file whose
  nav block no longer has `~` descriptions. Works for both single-file and bulk
  (`update .`) invocations.

### Fixed

- **`agentmap generate` heading offset** — files whose YAML frontmatter `---` is
  followed by a blank line had every heading line number written 1 too low. `separatorLines`
  now correctly detects whether a blank already exists at the insertion boundary.
- **Task list stale line numbers** — directed agents to run `agentmap update .` before
  starting description rewrites when `agentmap check` reports mismatches (affects projects
  indexed before the generate frontmatter fix).
- **`index-fixture` nav blocks** — corrected stale line numbers in four fixture files
  and added missing H1 nav entries that `generate` now includes.

### Changed

- **Task list format** — restructured `index-tasks.md` for small-model clarity: leads
  with a concise 5-step workflow and a before/after example; embeds the actual nav block
  in each task entry; moves the full nav-writing-guide to an appendix; collapses the
  two-checkbox-per-file format to a single checkbox.
- **Post-index instructions** — `agentmap index` output now says `Run agentmap next`
  rather than listing manual per-file steps.
- **Task list preamble** — updated "Your job" section to describe the `agentmap next`
  loop instead of the old `agentmap update <file>` + manual checkbox workflow.
- **Guide appendix** — task list appendix now uses `RulesContent` (sections 3–7 only)
  so the workflow/scratch sections already covered by the task list header are not
  duplicated; `agentmap guide` still shows the full guide unchanged.

### Infrastructure

- Added `internal/next` package (303 lines) with `FindTaskList`, `Next`, `FlushState`,
  `WriteState`, `RenderPrompt`, `RenderDone`, `RenderBlocked`.
- 542-line test suite for `internal/next` covering all state transitions.
- 112-line regression tests for the `generate` frontmatter offset fix.

---

## [v0.1.0-rc.6] — Post-index agent workflow improvements — 2026-04-09

Two targeted fixes to the post-`index` experience: a false-failure bug in
`agentmap check` that made every file look out of sync, and a ready-to-paste
agent prompt printed at the end of `agentmap index` to guide the description
review workflow.

### Fixed

- `agentmap check` reported false line-range mismatches for the last section in
  every POSIX file. `strings.Split` on a trailing-newline file produces a
  spurious empty trailing element, so `len(lines)` overcounted `totalLines` by 1.
  Fixed by using `strings.Count(content, "\n")` — consistent with `generate`
  and `update`, and equal to `wc -l`. Regression test added.

### Added

- `agentmap index` now prints a ready-to-paste agent prompt after writing the
  task list. The prompt directs the agent to read `.agentmap/index-tasks.md`
  for instructions, rewrite every `~`-prefixed `purpose` and `about` value,
  add `see` entries for closely related files, run `agentmap update <file>`
  after each edit, and finish with `agentmap check`.

---

## [v0.1.0-rc.5] — Budget-first nav pruning and embedded guide — 2026-04-08

This release replaces the old per-section threshold branching in nav generation
with a cleaner include-all-then-prune algorithm, adds the `agentmap guide`
command for in-shell reference, and fixes a critical bug in `update` that was
silently dropping new h3 sections.

### Added

- **`agentmap guide` command** — prints the nav-writing guide to stdout for quick
  in-shell reference. The guide is now embedded in the binary via `internal/guide`
  (Go `embed`), so it is always in sync with the installed version.
- **`agentmap index` embeds guide in task list** — `index-tasks.md` now includes
  the full nav-writing guide as a preamble so agents have all instructions in a
  single file without a separate lookup.

### Fixed

- **Critical: `update` was silently dropping new h3 sections** — the
  `expandThreshold` guard in `update.go` was gating inclusion of h3 entries on
  parent section size. Since `update` never knows the final budget, this caused
  newly added h3 headings to disappear from nav blocks after the first `update`
  run. Guard removed; `update` now preserves all entries unconditionally and lets
  `generate` (with `PruneNavEntries`) enforce the budget.
- **Hint trimming** — subsection hint text is now `TrimSpace`d before being
  appended to parent `about` fields, eliminating stray leading/trailing whitespace
  in generated hints.

### Changed

- **Budget-first nav generation** — `buildNavEntries` is now a flat loop that
  includes all h1–h3 sections unconditionally. The new `PruneNavEntries` function
  then enforces `max_nav_entries` by iteratively removing entries depth-first,
  classifying by parent section size: small parents (< `sub_threshold`) are
  silently dropped; medium parents (`sub_threshold`–`expand_threshold`) are
  collapsed to `>` hints; large parents (≥ `expand_threshold`) are unkillable.
  This replaces the previous `FilterNavEntries` approach.
- **`max_depth` default raised from 3 to 6** — the old value was used as a budget
  proxy; with budget-first pruning it is now a true sanity ceiling matching the
  Markdown heading maximum.
- **`FilterNavEntries` and `NavStubWords` removed** — retired concepts no longer
  needed under the budget-first model.
- **Nav-writing guide moved to `internal/guide/nav-writing-guide.md`** —
  `docs/nav-writing-guide.md` is now a symlink for discoverability; the canonical
  source is embedded in the binary.

### Infrastructure

- `testdata/README.md` added — documents each fixture's purpose, which tests use
  it, and the rule against modifying fixtures in place.
- `testdata/pruning-over-budget.md` added — static fixture that exercises all
  three `PruneNavEntries` branches (unkillable/hintable/droppable) under default
  config; 35 entries pruned to budget of 20.
- Stale `testdata/agentmap-design-expected.md` removed (unreferenced artifact).

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
