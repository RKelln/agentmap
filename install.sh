#!/usr/bin/env sh
# install.sh — install agentmap on Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/RKelln/agentmap/main/install.sh | sh
# Or:    curl -fsSL https://raw.githubusercontent.com/RKelln/agentmap/main/install.sh | sh -s -- --bin-dir /usr/local/bin

set -eu

REPO="RKelln/agentmap"
BIN_NAME="agentmap"
BIN_DIR="${BIN_DIR:-}"
VERSION="${VERSION:-latest}"
YES="${YES:-0}"

# --- color helpers ---
if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
  BOLD="$(tput bold 2>/dev/null || printf '')"
  GREEN="$(tput setaf 2 2>/dev/null || printf '')"
  YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
  RED="$(tput setaf 1 2>/dev/null || printf '')"
  RESET="$(tput sgr0 2>/dev/null || printf '')"
else
  BOLD=""
  GREEN=""
  YELLOW=""
  RED=""
  RESET=""
fi

info()  { printf '%s%s%s\n' "${GREEN}" "$*" "${RESET}"; }
warn()  { printf '%s%sWARN:%s %s\n' "${BOLD}" "${YELLOW}" "${RESET}" "$*" >&2; }
error() { printf '%s%sERROR:%s %s\n' "${BOLD}" "${RED}" "${RESET}" "$*" >&2; exit 1; }

# --- flag parsing ---
while [ $# -gt 0 ]; do
  case "$1" in
    --yes|-y)     YES=1 ;;
    --version)    shift; VERSION="$1" ;;
    --bin-dir)    shift; BIN_DIR="$1" ;;
    --help|-h)
      printf 'Usage: install.sh [--yes] [--version <ver>] [--bin-dir <dir>]\n'
      exit 0
      ;;
    *) error "Unknown flag: $1" ;;
  esac
  shift
done

# --- OS detection ---
detect_os() {
  os="$(uname -s)"
  case "$os" in
    Linux*)       printf 'Linux' ;;
    Darwin*)      printf 'Darwin' ;;
    MINGW*|MSYS*) error "Windows detected. Use install.ps1 instead." ;;
    *)            error "Unsupported OS: $os" ;;
  esac
}

# --- arch detection ---
detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  printf 'x86_64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *)             error "Unsupported architecture: $arch" ;;
  esac
}

# --- resolve default bin_dir ---
resolve_bin_dir() {
  if [ -n "$BIN_DIR" ]; then
    printf '%s' "$BIN_DIR"
    return
  fi
  # Prefer writable dir already in PATH
  for dir in /usr/local/bin "$HOME/.local/bin" "$HOME/bin"; do
    if [ -d "$dir" ] && [ -w "$dir" ]; then
      printf '%s' "$dir"
      return
    fi
  done
  # Fall back to ~/.local/bin (create it)
  printf '%s/.local/bin' "$HOME"
}

# --- check required tools ---
need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    error "Required command not found: $1"
  fi
}

need_cmd curl
need_cmd tar
if command -v sha256sum >/dev/null 2>&1; then
  :
elif command -v shasum >/dev/null 2>&1; then
  :
else
  error "Required command not found: sha256sum or shasum"
fi

# --- checksum verification ---
verify_checksum() {
  archive="$1"
  archive_name="$(basename "$archive")"
  checksum_file="$2"
  expected=$(grep "[[:space:]]${archive_name}$" "$checksum_file" | head -1 | cut -d' ' -f1 || true)
  if [ -z "$expected" ]; then
    error "Checksum for $archive_name not found in checksums.txt"
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$archive" | cut -d' ' -f1)
  else
    actual=$(shasum -a 256 "$archive" | cut -d' ' -f1)
  fi

  if [ "$expected" != "$actual" ]; then
    error "Checksum mismatch for $archive\n  expected: $expected\n  actual:   $actual"
  fi
}

# --- resolve latest version ---
resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    info "Fetching latest release..."
    if release_json=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null); then
      VERSION=$(printf '%s' "$release_json" | grep '"tag_name"' | head -1 | cut -d'"' -f4 || true)
    else
      warn "No stable release found; checking latest prerelease..."
      release_json=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases?per_page=1" 2>/dev/null || true)
      VERSION=$(printf '%s' "$release_json" | grep '"tag_name"' | head -1 | cut -d'"' -f4 || true)
    fi
    if [ -z "$VERSION" ]; then
      error "Failed to fetch latest version from GitHub API. Try --version vX.Y.Z"
    fi
  fi
  printf '%s' "$VERSION"
}

# --- main ---
OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="$(resolve_version)"
BIN_DIR="$(resolve_bin_dir)"

ARCHIVE="${BIN_NAME}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
DOWNLOAD_URL="${BASE_URL}/${ARCHIVE}"
CHECKSUM_URL="${BASE_URL}/checksums.txt"

printf '\n%s%sInstalling agentmap %s%s\n' "${BOLD}" "${GREEN}" "$VERSION" "${RESET}"
printf '  OS/Arch:  %s/%s\n' "$OS" "$ARCH"
printf '  Archive:  %s\n' "$ARCHIVE"
printf '  Install:  %s/%s\n\n' "$BIN_DIR" "$BIN_NAME"

# Confirm if interactive and --yes not given.
if [ "$YES" = "0" ] && [ -t 0 ]; then
  printf 'Proceed with installation? [y/N] '
  read -r answer
  case "$answer" in
    y|Y) ;;
    *) info "Installation cancelled."; exit 0 ;;
  esac
fi

# Create temp dir with cleanup trap.
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM

info "Downloading $ARCHIVE..."
curl -fsSL -o "${TMP}/${ARCHIVE}" "$DOWNLOAD_URL" \
  || error "Download failed: $DOWNLOAD_URL"

info "Verifying checksum..."
curl -fsSL -o "${TMP}/checksums.txt" "$CHECKSUM_URL" \
  || error "Failed to download checksums: $CHECKSUM_URL"
verify_checksum "${TMP}/${ARCHIVE}" "${TMP}/checksums.txt"

info "Extracting..."
tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP" "$BIN_NAME" \
  || error "Extraction failed"

# Create bin_dir if needed.
if [ ! -d "$BIN_DIR" ]; then
  mkdir -p "$BIN_DIR" || error "Failed to create $BIN_DIR"
fi

# Install the binary.
DEST="${BIN_DIR}/${BIN_NAME}"
if [ -w "$BIN_DIR" ]; then
  mv "${TMP}/${BIN_NAME}" "$DEST"
  chmod 755 "$DEST"
else
  info "Requesting sudo to install to $BIN_DIR..."
  sudo mv "${TMP}/${BIN_NAME}" "$DEST"
  sudo chmod 755 "$DEST"
fi

info "agentmap $VERSION installed to $DEST"

# PATH hint if needed.
case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *)
    warn "$BIN_DIR is not in your PATH."
    warn "Add this to your shell profile:"
    warn "  export PATH=\"\$PATH:${BIN_DIR}\""
    ;;
esac

printf '\n%s%sDone! Run: agentmap --help%s\n\n' "${BOLD}" "${GREEN}" "${RESET}"
