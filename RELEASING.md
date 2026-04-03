# Releasing agentmap

How to cut a release. GoReleaser handles builds, archives, checksums, GitHub
release creation, Homebrew formula push, and Scoop manifest push automatically
when a version tag is pushed.

## Agentic release (preferred)

Use the `/release` slash command in OpenCode:

```
/release v0.2.0 "Short description"
```

The command runs pre-flight checks, drafts `CHANGELOG.md` from git history,
waits for your approval, then tags and pushes — triggering GoReleaser
automatically. See `.opencode/commands/release.md` for the full workflow.

## Prerequisites (one-time setup)

1. **Create the Homebrew tap repo** — `RKelln/homebrew-agentmap` must exist as
   an empty public GitHub repo. GoReleaser pushes the formula here on every
   release. *(Already created.)*

   ```bash
   gh repo create RKelln/homebrew-agentmap --public \
     --description "Homebrew tap for agentmap"
   ```

2. **Create the Scoop bucket repo** — `RKelln/scoop-agentmap` must exist as an
   empty public GitHub repo. GoReleaser pushes the manifest here on every
   release. *(Create before the first release.)*

   ```bash
   gh repo create RKelln/scoop-agentmap --public \
     --description "Scoop bucket for agentmap"
   ```

3. **`GITHUB_TOKEN` permissions** — The default `GITHUB_TOKEN` in GitHub
   Actions has `contents: write` (set in `release.yml`). This is sufficient for
   creating releases and pushing to repos in the same account. No additional
   secrets needed.

## Manual release (fallback)

```bash
# 1. Ensure main is clean and CI passes
git checkout main && git pull
scripts/agent-run.sh make ci
goreleaser check

# 2. Update CHANGELOG.md with the new version entry, then commit
git add CHANGELOG.md
git commit -m "chore(release): add vX.Y.Z changelog entry"

# 3. Tag the release (semver; must start with v; use annotated tags)
git tag -a vX.Y.Z -m "vX.Y.Z — Short description"

# 4. Push commit and tag — tag push triggers the release workflow
git push origin main
git push origin vX.Y.Z
```

## What GoReleaser does on tag push

The `release.yml` workflow triggers on any `v*` tag and:

- Builds binaries for linux/darwin/windows × amd64/arm64
- Strips debug info (`-s -w`) and host paths (`-trimpath`) for reproducible builds
- Injects version and commit SHA via ldflags (`main.version`, `main.commit`)
- Creates archives: `.tar.gz` for Linux/macOS; `.zip` for Windows
- Generates `checksums.txt` (SHA256 of all archives)
- Creates a GitHub Release with all artifacts attached
- Pushes an updated `agentmap.rb` formula to `RKelln/homebrew-agentmap`
- Pushes an updated `agentmap.json` manifest to `RKelln/scoop-agentmap`

Monitor the workflow:

```bash
gh run watch --repo RKelln/agentmap \
  $(gh run list --repo RKelln/agentmap --workflow=release.yml \
    --limit=1 --json databaseId --jq '.[0].databaseId')
```

## Verifying locally (without publishing)

Requires [GoReleaser](https://goreleaser.com/install/) installed locally.

```bash
# Validate config
goreleaser check

# Build snapshot (all platforms; no publish; output to dist/)
goreleaser build --snapshot --clean
```

## Testing a release

Three levels of pre-release testing, in increasing effort:

### 1. Binary smoke test (fast, no extra tools)

Exercises the compiled binary against real testdata — catches ldflags
injection failures, embedded asset problems, and CLI issues that unit tests
(which use the package API) cannot. Also tests the upgrade pre-checks
(Homebrew/Scoop detection, dev-build rejection):

```bash
make smoke
```

`make smoke` includes an `upgrade --check` step that makes one real GitHub API
call. It will say "no releases found" before v0.1.0 is tagged, which is
expected and non-fatal.

To test a GoReleaser snapshot build instead of the local dev binary:

```bash
make smoke BINARY=dist/agentmap_Linux_x86_64_v1/agentmap
```

### 2. GoReleaser snapshot (catch packaging issues)

Runs the full GoReleaser pipeline locally without publishing — builds all 6
platform binaries, creates archives, generates checksums:

```bash
goreleaser release --snapshot --clean
# Then smoke test the Linux build:
make smoke BINARY=dist/agentmap_Linux_x86_64_v1/agentmap
```

### 3. Install script smoke test (requires Docker)

Tests `install.sh` end-to-end in a clean Ubuntu container — the closest thing
to a real user install:

```bash
make smoke-install
```

This pulls `install.sh` from the `main` branch on GitHub, so it only works
after changes to the script are already pushed.

### 4. Pre-release tag

Push a release candidate tag to trigger the full GoReleaser workflow without
polluting the stable release channel:

```bash
git tag -a v0.1.0-rc.1 -m "v0.1.0-rc.1 release candidate"
git push origin v0.1.0-rc.1
```

Then verify the GitHub Release, Homebrew formula, and Scoop manifest are
created correctly before tagging the final release.

## Installing from a release

```bash
# Homebrew (macOS/Linux)
brew install RKelln/agentmap/agentmap

# Scoop (Windows)
scoop bucket add agentmap https://github.com/RKelln/scoop-agentmap
scoop install agentmap

# Shell script (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/RKelln/agentmap/main/install.sh | sh

# PowerShell (Windows)
irm https://raw.githubusercontent.com/RKelln/agentmap/main/install.ps1 | iex

# go install (any platform with Go installed)
go install github.com/RKelln/agentmap/cmd/agentmap@latest
```

## Upgrade path

`agentmap upgrade` self-updates the binary in place using the GitHub Releases
checksums for verification. There are no migrations to run on upgrade — agentmap
has no database, no versioned state files, and no persistent cache. Upgrades are
safe atomic binary replacement.

**Install-method behaviour:**

| Install method | `agentmap upgrade` behaviour |
|---|---|
| Direct (`install.sh`, `install.ps1`) | Downloads and replaces binary |
| `go install` | Downloads and replaces binary |
| Homebrew | Refuses — prints `brew upgrade agentmap` |
| Scoop | Refuses — prints `scoop update agentmap` |

**Testing the upgrade path** (requires a real release to exist):

```bash
# Check for updates without downloading
agentmap upgrade --check

# Full upgrade (direct installs only)
agentmap upgrade

# Verify after upgrade
agentmap version
```

## Changelog

`CHANGELOG.md` in the repo root follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format. Update it
before tagging — the `/release` command does this automatically.

The GoReleaser GitHub Release notes are auto-generated from git log and exclude
commits prefixed with `docs:`, `test:`, or `chore:`. Use conventional commit
types (`feat:`, `fix:`, `refactor:`) for changes that should appear there.

