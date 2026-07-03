#!/bin/sh
# install.sh - install machinery without cloning the repo.
#
# Fetches the machinery CLI binary and the agent skill + role docs from a
# GitHub release and installs them into your agent home(s). No git checkout,
# no Go toolchain, no Python.
#
#   curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
#
# Environment overrides (all optional):
#   MACHINERY_VERSION      release tag to install, or "latest" (default: latest)
#   MACHINERY_HOMES        space-separated agent homes; the FIRST holds the real
#                          files, the rest are symlinked to it
#                          (default: "$HOME/.agents $HOME/.claude")
#   INSTALL_DIR            where the CLI binary lands (default: "$HOME/.local/bin")
#   MACHINERY_REPO         owner/name to fetch from (default: RamXX/machinery)
#   MACHINERY_SKILL_SRC    use this local dir as the skill/agents source instead
#                          of downloading (expects skills/ and agents/ under it)
#   MACHINERY_SKIP_BINARY  set to 1 to skip installing the CLI binary
set -eu

REPO="${MACHINERY_REPO:-RamXX/machinery}"
VERSION="${MACHINERY_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
HOMES="${MACHINERY_HOMES:-$HOME/.agents $HOME/.claude}"

say() { printf '%s\n' "$*"; }
die() { printf 'install: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "required tool not found: $1"; }

need curl
need tar

# --- detect os/arch (must match the release asset names) -------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported architecture: $arch" ;;
esac
ext=""
case "$os" in
  linux|darwin) ;;
  msys*|mingw*|cygwin*|windows*) os=windows; ext=".exe" ;;
  *) die "unsupported OS: $os" ;;
esac
binname="machinery"
if [ "$os" = windows ]; then binname="machinery.exe"; fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/machinery.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

sha256() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
  else die "no sha256 tool (shasum or sha256sum) found"; fi
}

# --- resolve the release tag -----------------------------------------------
if [ "$VERSION" = "latest" ]; then
  curl -fsSL -o "$tmp/rel.json" "https://api.github.com/repos/$REPO/releases/latest" \
    || die "cannot reach the GitHub API to resolve the latest release"
  TAG=$(grep '"tag_name"' "$tmp/rel.json" | head -1 |
    sed 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/')
  [ -n "${TAG:-}" ] || die "no published release found for $REPO"
else
  TAG="$VERSION"
fi
say "machinery $TAG ($os/$arch)"

# --- 1. install the CLI binary ---------------------------------------------
if [ "${MACHINERY_SKIP_BINARY:-0}" = "1" ]; then
  say "skipping CLI binary (MACHINERY_SKIP_BINARY=1)"
else
  asset="machinery-${os}-${arch}${ext}"
  base="https://github.com/$REPO/releases/download/$TAG"
  say "Downloading $asset..."
  curl -fsSL -o "$tmp/$asset" "$base/$asset" || die "failed to download $asset from $TAG"
  curl -fsSL -o "$tmp/checksums-sha256.txt" "$base/checksums-sha256.txt" 2>/dev/null \
    || die "release $TAG has no checksums-sha256.txt; refusing to install an unverified binary"
  want=$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums-sha256.txt")
  got=$(sha256 "$tmp/$asset")
  [ -n "$want" ] || die "no checksum listed for $asset"
  [ "$want" = "$got" ] || die "checksum mismatch for $asset (want $want, got $got)"
  say "checksum verified"
  mkdir -p "$INSTALL_DIR"
  cp "$tmp/$asset" "$INSTALL_DIR/$binname"
  chmod +x "$INSTALL_DIR/$binname"
  say "installed $binname -> $INSTALL_DIR/$binname"
fi

# --- 2. obtain the skill + agent sources -----------------------------------
if [ -n "${MACHINERY_SKILL_SRC:-}" ]; then
  src="$MACHINERY_SKILL_SRC"
  [ -d "$src/skills/machinery" ] || die "MACHINERY_SKILL_SRC has no skills/machinery: $src"
else
  say "Fetching skill + agents..."
  curl -fsSL -o "$tmp/src.tar.gz" "https://github.com/$REPO/archive/refs/tags/$TAG.tar.gz" \
    || die "failed to download source tarball for $TAG"
  mkdir -p "$tmp/src"
  tar -xzf "$tmp/src.tar.gz" -C "$tmp/src"
  # the archive unpacks into a single top-level dir (e.g. machinery-0.1.1/)
  src=$(find "$tmp/src" -maxdepth 1 -mindepth 1 -type d | head -1)
  { [ -n "$src" ] && [ -d "$src/skills/machinery" ]; } || die "unexpected tarball layout"
fi

# --- 3. lay out canonical files + symlinks ---------------------------------
# The first home in $HOMES holds the real files; the rest symlink to it.
canon=""
for home in $HOMES; do
  if [ -z "$canon" ]; then
    canon="$home"
    mkdir -p "$canon/skills" "$canon/agents"
    rm -rf "$canon/skills/machinery"
    cp -R "$src/skills/machinery" "$canon/skills/machinery"
    cp "$src/agents/machinery-fsm-author.md" "$src/agents/machinery-build-writer.md" "$canon/agents/"
    say "installed skill + agents -> $canon (canonical)"
  else
    mkdir -p "$home/skills" "$home/agents"
    rm -rf "$home/skills/machinery"
    ln -sfn "$canon/skills/machinery" "$home/skills/machinery"
    ln -sfn "$canon/agents/machinery-fsm-author.md" "$home/agents/machinery-fsm-author.md"
    ln -sfn "$canon/agents/machinery-build-writer.md" "$home/agents/machinery-build-writer.md"
    say "linked skill + agents -> $home (-> $canon)"
  fi
done

# --- 4. best-effort environment check --------------------------------------
mach="$INSTALL_DIR/$binname"
if [ -x "$mach" ]; then
  "$mach" preflight || true
elif command -v machinery >/dev/null 2>&1; then
  machinery preflight || true
fi
case ":${PATH}:" in
  *":$INSTALL_DIR:"*) : ;;
  *)
    if [ "${MACHINERY_SKIP_BINARY:-0}" != "1" ]; then
      say ""
      say "note: $INSTALL_DIR is not on your PATH. Add it:"
      say "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    ;;
esac
