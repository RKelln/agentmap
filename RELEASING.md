# Releasing agentmap

How to cut a release. GoReleaser handles builds; archives; checksums; GitHub release creation; and Homebrew formula push automatically when a version tag is pushed.

## Prerequisites (one-time setup)

1. **Create the Homebrew tap repo** -- `ryankelln/homebrew-agentmap` must exist on GitHub as an empty public repo. GoReleaser writes the formula to it on every release. Without it the release workflow fails.

   ```bash
   gh repo create ryankelln/homebrew-agentmap --public --description "Homebrew tap for agentmap"
   ```

2. **`GITHUB_TOKEN` permissions** -- The default `GITHUB_TOKEN` in GitHub Actions has `contents: write` (set in `release.yml`). This is sufficient for creating releases and pushing to repos in the same account. No additional secrets needed.

## Cutting a release

```bash
# 1. Ensure main is clean and CI passes
git checkout main && git pull
scripts/agent-run.sh make ci

# 2. Tag the release (semver; must start with v)
git tag v0.1.0 -m "Release v0.1.0"

# 3. Push the tag -- this triggers the release workflow
git push origin v0.1.0
```

The `release.yml` workflow runs on the tag push and:
- Builds binaries for linux/darwin/windows × amd64/arm64
- Strips debug info (`-s -w`) and host paths (`-trimpath`) for reproducible builds
- Injects version and commit SHA via ldflags
- Creates archives: `.tar.gz` for Linux/macOS; `.zip` for Windows
- Generates `checksums.txt` (SHA256 of all archives)
- Creates a GitHub Release with all artifacts attached
- Pushes an updated `agentmap.rb` formula to `ryankelln/homebrew-agentmap`

## Verifying a release locally (without publishing)

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
brew install ryankelln/tap/agentmap

# go install (any platform with Go installed)
go install github.com/ryankelln/agentmap/cmd/agentmap@latest

# Direct download (Linux/macOS)
curl -fsSL https://github.com/ryankelln/agentmap/releases/latest/download/agentmap_Linux_x86_64.tar.gz | tar xz
```

## Changelog filtering

The GoReleaser changelog excludes commits with these prefixes: `docs:` `test:` `chore:`. Use conventional commit types (`feat:` `fix:` `refactor:`) for changes that should appear in the release notes.
