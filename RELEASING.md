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
go install github.com/ryankelln/agentmap/cmd/agentmap@latest
```

## Changelog

`CHANGELOG.md` in the repo root follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format. Update it
before tagging — the `/release` command does this automatically.

The GoReleaser GitHub Release notes are auto-generated from git log and exclude
commits prefixed with `docs:`, `test:`, or `chore:`. Use conventional commit
types (`feat:`, `fix:`, `refactor:`) for changes that should appear there.

