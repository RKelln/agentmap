# agentmap Implementation Tasks

Vertical slices -- each phase delivers a working CLI command with tests.

## Phase 1: `generate` -- Headings to Nav Block (v0.1)

**Goal:** `agentmap generate ./docs` produces nav blocks with empty descriptions.

**What's already done:**
- `parser.ParseHeadings` -- extracts h1-h3, code-fence aware (14 tests)
- `navblock.ParseNavBlock` / `navblock.RenderNavBlock` -- round-trip (7 tests)
- Test fixtures in `testdata/`

**What's needed:**
1. **Section computation** -- given headings, compute `s,n` ranges for each section
   - Section ends at line before next heading at same or higher level, or EOF
   - Empty sections: `s == e`
2. **Nav block writer** -- insert/replace nav block in file content
   - After YAML frontmatter (if present), before first heading
   - Replace existing block in place
   - One blank line between nav block and first heading
3. **File discovery** -- find all `.md` files under a path (recursive)
   - Skip `.git/` directory
   - v0.1: no `.gitignore` parsing needed, just skip `node_modules`, `.git`
4. **`generate` command** -- wire it all together
   - For each file: parse headings → compute sections → render nav block → write file
   - `--dry-run` flag: print without writing
   - Output: `Generated: path (N sections)` / `Skipped: path (under N lines; purpose-only)`
   - Purpose-only blocks for files under `min_lines` (default 50)

**Tests:** section computation, nav block insertion, file discovery, generate command integration with testdata fixtures.

---

## Phase 2: `generate` -- Keyword Descriptions

**Goal:** `agentmap generate` produces nav blocks with keyword-based descriptions.

**What's needed:**
1. **Keyword extraction** -- TF-IDF per section
   - Tokenize, lowercase, remove stopwords, remove short tokens
   - Term frequency within section
   - Optional IDF against other sections in same document
   - Top 4-6 terms joined by semicolons
2. **Purpose generation** -- keywords from first paragraph or whole file
3. **Subsection hints** -- `>` suffix with abbreviated h3 names
   - Three-tier threshold: no hints (<50 lines), hints (50-150), full h3 entries (>150)
4. **Flags** -- `--min-lines`, `--sub-threshold`, `--expand-threshold`

**Tests:** keyword extraction (stopwords, TF, IDF, edge cases), purpose generation, threshold logic, integration with generate.

---

## Phase 3: `update` -- Line Number Refresh

**Goal:** `agentmap update ./docs` refreshes line numbers, preserves descriptions.

**What's needed:**
1. **Nav matching** -- match existing nav entries to current headings
   - Strip `#` prefixes from `name`, match by text
   - Duplicate headings: match in order of appearance
   - New headings: add with empty `about`
   - Deleted headings: remove entry
2. **Line number update** -- recompute `s,n` for matched headings
3. **Git diff integration** -- `git diff HEAD -- <file>` → changed line ranges
   - Map changed ranges to sections
   - Mark sections as `content-changed` or `new`
4. **Report output** -- the primary interface for agents
   - `OK`, `shifted`, `content-changed`, `new`, `deleted`
   - Current description text for changed sections
5. **Flags** -- `--quiet`, `--dry-run`
6. **Exit codes** -- 0 success, 1 error

**Tests:** nav matching (rename, new, delete, duplicate), line update, git diff mapping, report formatting, integration with modified testdata.

---

## Phase 4: `check` -- Validation

**Goal:** `agentmap check ./docs` validates nav blocks are in sync.

**What's needed:**
1. **Validation** -- parse nav block, reparse headings, compare
   - Mismatched line numbers
   - Headings in nav but not in document
   - Headings in document but not in nav
2. **Report** -- `FAIL: path` with details per mismatch
3. **Exit codes** -- 0 all sync, 1 one or more failures
4. **Never modifies files**

**Tests:** validation logic, all mismatch types, exit codes, integration.

---

## Phase 5: Configuration + File Discovery

**Goal:** `agentmap.yml` config + proper `.gitignore`-aware file discovery.

**What's needed:**
1. **Config loader** -- read `agentmap.yml`, apply defaults
   - `min_lines`, `sub_threshold`, `expand_threshold`, `max_depth`, `exclude`
   - Config overrides via CLI flags take precedence
2. **File discovery** -- `git ls-files` when in a git repo
   - Respects `.gitignore` automatically
   - Filter against `exclude` list from config
   - Fallback to `filepath.Walk` when not in a git repo
3. **Edge cases** -- nav block corruption handling, empty sections, special characters in headings, very large files (cap at ~20 entries)

**Tests:** config loading (missing file, partial config, overrides), file discovery (git repo, non-git repo, exclude patterns), edge cases.

---

## Phase 6: Polish + Pre-Commit Hook

**Goal:** Production-ready tool with pre-commit hook support.

**What's needed:**
1. **Pre-commit hook template** -- `.git/hooks/pre-commit` script
2. **Performance** -- `update` and `check` under 100ms for 50 files
3. **Error handling** -- clear error messages, `fmt.Errorf("context: %w", err)`
4. **`--llm` flag stub** -- flag exists but returns error if not configured (LLM integration is out of scope for v0.1)

**Tests:** hook template, performance benchmark, error paths.

---

## Phase 7: Distribution + Agent Onboarding

**Goal:** Zero-friction install, upgrade, and per-project agent setup. A user (or their agent) should go from "never heard of agentmap" to "fully configured" in one command.

### 7.1 GoReleaser + Release Workflow

Set up automated multi-platform binary releases.

**What's needed:**
1. **`.goreleaser.yaml`** -- build matrix for linux/darwin/windows x amd64/arm64
   - `CGO_ENABLED=0` for static binaries
   - Version injection via ldflags: `-X main.version={{.Version}} -X main.commit={{.Commit}}`
   - Archive naming: `agentmap_{Os}_{Arch}.tar.gz` (`.zip` for Windows)
   - `checksums.txt` generation
   - Homebrew tap formula pushed to `ryankelln/homebrew-agentmap`
2. **`.github/workflows/release.yml`** -- trigger on `v*` tags
   - `goreleaser/goreleaser-action@v7`
   - Permissions: `contents: write`
3. **Version variable** -- ensure `main.version` is set via ldflags in both Makefile and GoReleaser
   - `agentmap version` subcommand (or `--version` flag) prints it

**Tests:** `agentmap version` outputs the injected version string.

### 7.2 Install Scripts -- Curl + PowerShell Installers

One-line installers for Unix and Windows.

**What's needed:**
1. **`install.sh`** at repo root -- POSIX sh; no bash-isms (Linux/macOS)
   - Detect OS via `uname -s` (Linux/Darwin)
   - Detect arch via `uname -m` (x86_64 -> amd64; aarch64 -> arm64)
   - Download from GitHub Releases (`latest` or `$VERSION`)
   - Verify checksum against `checksums.txt`
   - Extract to `$BIN_DIR` (default `/usr/local/bin`); `sudo` only if needed
   - `--yes` flag to skip confirmation prompt (for CI / piped usage)
   - `--version` flag to pin a specific release
   - `--bin-dir` flag to override install location
   - Colorized output with `tput` (graceful fallback)
   - Clean up temp files on exit (trap EXIT/INT/TERM)
2. **`install.ps1`** at repo root -- PowerShell (Windows)
   - Detect arch via `$env:PROCESSOR_ARCHITECTURE` (AMD64 -> amd64; ARM64 -> arm64)
   - Download `.zip` from GitHub Releases
   - Verify checksum against `checksums.txt`
   - Extract to `$BinDir` (default `$env:LOCALAPPDATA\agentmap`); add to user PATH if not present
   - `-Yes` flag to skip confirmation
   - `-Version` flag to pin a specific release
   - `-BinDir` flag to override install location
   - Clean up temp files in `finally` block
3. **Usage:**
   ```
   # Linux / macOS
   curl -sSfL https://raw.githubusercontent.com/ryankelln/agentmap/main/install.sh | sh
   curl -sSfL https://raw.githubusercontent.com/ryankelln/agentmap/main/install.sh | sh -s -- --version v0.1.0

   # Windows (PowerShell)
   irm https://raw.githubusercontent.com/ryankelln/agentmap/main/install.ps1 | iex
   # Or with options:
   & ([scriptblock]::Create((irm https://raw.githubusercontent.com/ryankelln/agentmap/main/install.ps1))) -Version v0.1.0
   ```

**Tests:** shellcheck passes on `install.sh`; PSScriptAnalyzer passes on `install.ps1`; both tested in CI (Linux amd64 + Darwin arm64 + Windows amd64).

### 7.3 `agentmap upgrade` -- Self-Update Command

**What's needed:**
1. **`creativeprojects/go-selfupdate`** dependency -- checks GitHub Releases API; matches OS/arch to asset name; downloads; replaces current binary
2. **`upgrade` cobra command:**
   - Compare `main.version` against latest GitHub release tag
   - If already latest: print "Already up to date (vX.Y.Z)" and exit 0
   - If newer available: download; verify checksum; replace binary in place
   - Print "Updated agentmap vX.Y.Z -> vX.Y.Z"
   - `--check` flag: only report whether an update is available (no download)
   - Handle permission errors gracefully: suggest `sudo agentmap upgrade` or reinstall
3. **Build-time version required** -- `upgrade` refuses to run if version is "dev" (local build without ldflags)

**Tests:** version comparison logic; `--check` flag output; graceful error on missing version.

### 7.4 `agentmap init` -- Project Onboarding

Safe, idempotent command that configures a project's agents to understand AGENT:NAV blocks.

**What's needed:**
1. **Core behavior:**
   - Detect which agent tool config files already exist in the repo
   - Append agentmap instructions to the appropriate file(s)
   - Never overwrite or replace existing content -- append only
   - Idempotent: if instructions are already present (detect marker comment); skip and report
   - Always show a diff/preview of what will be written; require confirmation (or `--yes`)
2. **Marker for idempotency:**
   ```
   <!-- agentmap:init -->
   ```
   Presence of this marker in a file means init already ran; skip that file.
3. **Instruction content** -- the "Reading Markdown Files" and "Before Committing Markdown Changes" blocks from agentmap-design.md section 7.1; adapted per tool:
   ```markdown
   <!-- agentmap:init -->
   ## Reading Markdown Files

   Every markdown file over 50 lines has an AGENT:NAV block in the first
   20 lines. Read it before reading the rest of the file.

   - If purpose doesn't match your task; stop reading.
    - Use s,n line ranges: Read(offset=s, limit=n) for the section you need.
   - Check see before searching; the file you need may be listed.

   ## Before Committing Markdown Changes

   1. Run: agentmap update <changed files>
   2. Review output for sections marked content-changed or new.
   3. Read flagged sections and update their descriptions in the nav block.
   4. Commit.
   ```
4. **Multi-tool detection and targets:**

   | File detected | Action |
   |---|---|
   | `AGENTS.md` | Append instructions (covers OpenCode; Copilot; Windsurf; Roo Code; Zed) |
   | `CLAUDE.md` | Append instructions (Claude Code) |
   | `.cursorrules` | Append instructions (Cursor legacy) |
   | `.cursor/rules/` dir exists | Write `.cursor/rules/agentmap.md` (Cursor modern) |
   | `.windsurf/rules/` dir exists | Write `.windsurf/rules/agentmap.md` with trigger frontmatter |
   | `.continue/rules/` dir exists | Write `.continue/rules/agentmap.md` |
   | `.roo/rules/` dir exists | Write `.roo/rules/agentmap.md` |
   | `.amazonq/rules/` dir exists | Write `.amazonq/rules/agentmap.md` |
   | `.aider.conf.yml` exists | Warn that user should add `read: [AGENTS.md]` (do not auto-edit YAML) |
   | None of the above | Create `AGENTS.md` with instructions (best cross-tool default) |

5. **Pre-commit hook setup:**
   - Detect existing hook infrastructure and offer to install `agentmap check`:

   | Hook system detected | Action |
   |---|---|
   | `.pre-commit-config.yaml` exists | Append agentmap hook entry to YAML (with confirmation) |
   | `.husky/` dir exists | Write `.husky/pre-commit` snippet (or append to existing) |
   | `.git/hooks/pre-commit` exists | Append `agentmap check` guard (with confirmation) |
   | `.lefthook.yml` exists | Warn: "Add agentmap check to your lefthook config manually" |
   | `.git/hooks/` only | Offer to create `.git/hooks/pre-commit` with `agentmap check` |
   | Not a git repo | Warn: "No git repo detected; skipping hook setup" |

   - Hook script content (for `.git/hooks/pre-commit` and `.husky/`):
     ```bash
     # agentmap: validate nav blocks before commit
     if command -v agentmap >/dev/null 2>&1; then
       agentmap check . || {
         echo "AGENT:NAV blocks are out of sync. Run: agentmap update ."
         exit 1
       }
     fi
     ```
   - `.pre-commit-config.yaml` entry:
     ```yaml
     - repo: local
       hooks:
         - id: agentmap-check
           name: Validate AGENT:NAV blocks
           entry: agentmap check
           language: system
           types: [markdown]
     ```
   - `--no-hook` flag to skip hook setup entirely
6. **Flags:**
   - `--yes` -- skip confirmation prompt
   - `--dry-run` -- show what would be written without writing
   - `--tool <name>` -- only install for a specific tool (e.g. `--tool cursor`)
   - `--all` -- install for all detected tools (default: only detected ones)
   - `--no-hook` -- skip pre-commit hook setup
7. **Output:**
   ```
   Detected: AGENTS.md, .cursor/rules/, .pre-commit-config.yaml
   
   Will append to: AGENTS.md (28 lines)
   Will create: .cursor/rules/agentmap.md (22 lines)
   Will append to: .pre-commit-config.yaml (hook entry)
   Skipped: .aider.conf.yml (manual config needed; add 'read: [AGENTS.md]')
   
   Proceed? [y/N]
   ```

**Safety rules:**
- Never delete or overwrite existing content
- Never modify YAML/JSON config files automatically (too risky) -- warn and instruct instead
- Exception: `.pre-commit-config.yaml` is safe to append a hook entry (it's a list)
- Always preview changes before writing
- Idempotency via marker comment; running twice is a no-op
- Hook idempotency: check for existing `agentmap check` in hook before adding

**Tests:** detection logic for each tool; idempotency (run twice; second is no-op); `--dry-run` output; marker detection; append-not-overwrite verification; hook detection for each hook system; hook idempotency.

### 7.5 Agent Skill Files (Cross-Tool)

Pre-built skill/rule files for tools that support directory-based rules. These ship with the agentmap binary and are written by `init`.

**What's needed:**
1. **Embedded skill templates** -- Go `embed` directive; one template per tool format
   - Plain markdown (AGENTS.md / CLAUDE.md / .cursorrules / .clinerules)
   - Cursor `.cursor/rules/agentmap.md` with frontmatter:
     ```yaml
     ---
     description: How to read and maintain AGENT:NAV blocks in markdown files
     globs: "**/*.md"
     alwaysApply: false
     ---
     ```
   - Windsurf `.windsurf/rules/agentmap.md` with frontmatter:
     ```yaml
     ---
     trigger: model_decision
     description: How to read and maintain AGENT:NAV blocks in markdown files
     ---
     ```
   - Continue `.continue/rules/agentmap.md` with frontmatter:
     ```yaml
     ---
     name: agentmap
     description: How to read and maintain AGENT:NAV blocks in markdown files
     globs: "**/*.md"
     ---
     ```
   - Roo Code `.roo/rules/agentmap.md` (plain markdown; no frontmatter)
   - Amazon Q `.amazonq/rules/agentmap.md` (plain markdown)
2. **Template variable substitution** -- none needed for v0.1; all templates are static
3. **OpenCode skill** -- `.opencode/skills/agentmap/SKILL.md` with proper frontmatter:
   ```yaml
   ---
   name: agentmap-maintain
   description: Read and maintain AGENT:NAV blocks in markdown files
   ---
   ```

**Tests:** embedded templates parse correctly; each template contains the marker comment; no commas in template text that would break nav block examples.

### 7.6 README Agent Section

Add an agent-oriented section to README.md so agents pointed at the repo can self-onboard.

**What's needed:**
1. **"For AI Agents" section** in README.md:
   - One-line: what agentmap does and why the agent should care
   - `agentmap init` as the single setup command
   - Link to AGENTS.md for full workflow details
   - Explicit: "If you are an AI coding agent; run `agentmap init` in the user's project"
2. **Install section update** -- add curl installer and Homebrew to the existing install section

**Tests:** none (documentation only).

### 7.7 Package Manager Support

**What's needed:**
1. **Homebrew tap** -- handled by GoReleaser (7.1); create `ryankelln/homebrew-agentmap` repo
2. **`go install`** -- already works; no changes needed
3. **Scoop bucket** (Windows) -- GoReleaser `scoops:` section; create `ryankelln/scoop-agentmap` repo
4. **Document all methods** in README install section:
   ```
   # Homebrew
   brew install ryankelln/tap/agentmap
   
   # Curl
   curl -sSfL https://raw.githubusercontent.com/ryankelln/agentmap/main/install.sh | sh
   
   # Go
   go install github.com/RKelln/agentmap/cmd/agentmap@latest
   
   # Upgrade
   agentmap upgrade
   ```

**Tests:** GoReleaser config validates (`goreleaser check`).

### 7.8 `agentmap uninit` + `agentmap uninstall` -- Clean Removal

**Goal:** Reverse everything `init` and the installers set up. No orphaned config fragments.

**What's needed:**

1. **`agentmap uninit`** -- removes project-level agent instructions and hooks
   - Scan all files that `init` would have written to (same detection logic as 7.4)
   - For files where content was appended (AGENTS.md; CLAUDE.md; .cursorrules):
     - Find the `<!-- agentmap:init -->` marker
     - Remove the marker and all agentmap-injected content that follows it (up to the next non-agentmap section or EOF)
     - If the file becomes empty after removal; delete it only if `init` created it (track via marker metadata or separate `.agentmap-state.json`)
   - For files `init` created outright (`.cursor/rules/agentmap.md`; `.roo/rules/agentmap.md`; etc.): delete them
   - Remove pre-commit hook additions:
     - `.pre-commit-config.yaml`: remove the `agentmap-check` hook entry
     - `.git/hooks/pre-commit`: remove the `agentmap check` guard block
     - `.husky/pre-commit`: remove the agentmap snippet
   - Always preview what will be removed; require confirmation (or `--yes`)
   - `--dry-run` flag: show what would be removed without writing
   - Idempotent: if no marker found; report "agentmap not initialized" and exit 0
   - **Output:**
     ```
     Will remove from: AGENTS.md (28 lines; agentmap instructions block)
     Will delete: .cursor/rules/agentmap.md
     Will remove from: .pre-commit-config.yaml (agentmap-check hook entry)
     
     Proceed? [y/N]
     ```

2. **`agentmap uninstall`** -- removes the agentmap binary itself
   - Runs `uninit` first (with confirmation) if the current directory has agentmap config
   - Locates the running binary via `os.Executable()`
   - Removes the binary file
   - Platform-aware messaging:
     - If installed via Homebrew: "Installed via Homebrew. Run: brew uninstall agentmap"
     - If installed via Scoop: "Installed via Scoop. Run: scoop uninstall agentmap"
     - If installed via `go install`: "Installed via go install. Run: go clean -i github.com/RKelln/agentmap/cmd/agentmap"
     - Otherwise: removes the binary directly (with sudo prompt if needed)
   - Detection heuristic for install method:
     - Binary path contains `/Cellar/` or `/homebrew/` -> Homebrew
     - Binary path contains `/scoop/` -> Scoop
     - Binary path under `$GOPATH/bin` or `$GOBIN` -> go install
     - Otherwise -> direct install
   - `--keep-config` flag: only remove the binary; leave project configs intact (skip `uninit`)
   - `--yes` flag: skip confirmation

**Safety rules:**
- `uninit` never touches nav blocks inside markdown files -- only the init-injected instructions
- `uninit` never deletes files it didn't create (uses marker to identify its content)
- `uninstall` via Homebrew/Scoop/go defers to those tools -- never force-removes a managed binary
- Both commands always preview before acting

**Tests:** uninit removes appended content without damaging surrounding file content; uninit is idempotent; uninit cleans up hook entries from all supported hook systems; uninstall detects install method correctly; `--dry-run` for both commands.

### 7.9 GitHub Action -- `agentmap-check` Reusable Action

**Goal:** Two lines of YAML in any repo's CI to get nav block validation on every PR.

**What's needed:**
1. **`action.yml`** at repo root (composite action) or in a dedicated `ryankelln/agentmap-action` repo
   - Inputs: `version` (default: `latest`); `path` (default: `.`); `args` (extra flags)
   - Steps: install agentmap (from GitHub Releases); run `agentmap check <path>`
   - Annotate failures as PR check annotations (use `::error file=...` workflow commands)
2. **Composite action implementation:**
   ```yaml
   name: "agentmap check"
   description: "Validate AGENT:NAV blocks are in sync with markdown headings"
   inputs:
     version:
       description: "agentmap version to install"
       default: "latest"
     path:
       description: "Path to check"
       default: "."
   runs:
     using: "composite"
     steps:
       - name: Install agentmap
         shell: bash
         run: |
           curl -sSfL https://raw.githubusercontent.com/ryankelln/agentmap/main/install.sh \
             | sh -s -- --yes --version ${{ inputs.version }}
       - name: Check nav blocks
         shell: bash
         run: agentmap check ${{ inputs.path }}
   ```
3. **Usage in consumer repos:**
   ```yaml
   # .github/workflows/ci.yml
   jobs:
     agentmap:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - uses: ryankelln/agentmap-action@v1
   ```
4. **PR comment on failure** (stretch goal) -- use `gh` or the GitHub API to post a comment with the `check` report and suggested fix (`agentmap update .`)

**Tests:** action runs successfully on a repo with valid nav blocks (exit 0); action fails on a repo with stale nav blocks (exit 1); install script works in GitHub Actions runner environment.
