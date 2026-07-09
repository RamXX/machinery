#!/bin/sh
# install.sh - bootstrap machinery without cloning the repo.
#
# Downloads the machinery CLI binary (checksum-verified) from a GitHub release,
# then hands off to `machinery install` to place the skill + role docs into your
# agent homes. All the placement logic lives in the binary; this script only
# has to deliver the first binary.
#
#   curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh
#
# Environment overrides (all optional):
#   MACHINERY_VERSION      release tag to install, or "latest" (default: latest)
#   MACHINERY_HOMES        space-separated agent homes; the FIRST is canonical.
#                          Unset, the binary uses its defaults ("$HOME/.agents"
#                          then "$HOME/.claude") and skips any home the Claude
#                          Code plugin already serves; setting this passes the
#                          homes explicitly, which always wins over that skip.
#   MACHINERY_TARGETS      space-separated host adapters from claude, codex,
#                          opencode, all. Cannot be combined with MACHINERY_HOMES.
#   INSTALL_DIR            where the CLI binary lands (default: "$HOME/.local/bin")
#   MACHINERY_REPO         owner/name to fetch from (default: RamXX/machinery)
#   MACHINERY_BIN          use this machinery binary instead of downloading (dev/test)
#   MACHINERY_SKILL_SRC    pass a local checkout to `machinery install --from` (offline)
set -eu

REPO="${MACHINERY_REPO:-RamXX/machinery}"
VERSION="${MACHINERY_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
HOMES="${MACHINERY_HOMES:-}"
TARGETS="${MACHINERY_TARGETS:-}"

say() { printf '%s\n' "$*"; }
die() { printf 'install: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "required tool not found: $1"; }

[ -z "$HOMES" ] || [ -z "$TARGETS" ] || die "MACHINERY_HOMES and MACHINERY_TARGETS cannot be combined"

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

# --- obtain the binary -----------------------------------------------------
if [ -n "${MACHINERY_BIN:-}" ]; then
  mach="$MACHINERY_BIN"
  say "using machinery binary: $mach"
else
  need curl
  sha256() {
    if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'
    elif command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
    else die "no sha256 tool (shasum or sha256sum) found"; fi
  }
  tmp=$(mktemp -d "${TMPDIR:-/tmp}/machinery.XXXXXX")
  trap 'rm -rf "$tmp"' EXIT INT TERM
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
  mach="$INSTALL_DIR/$binname"
  say "installed $binname -> $mach"
fi

# --- place the skill + role docs (the binary owns this) --------------------
# No --home flags unless MACHINERY_HOMES is set: the binary's default home
# list already skips any home the Claude Code plugin serves, and an explicit
# --home is the documented way to override that.
set -- install
for h in $HOMES; do
  set -- "$@" --home "$h"
done
for target in $TARGETS; do
  set -- "$@" --target "$target"
done
if [ -n "${MACHINERY_SKILL_SRC:-}" ]; then
  set -- "$@" --from "$MACHINERY_SKILL_SRC"
fi
if [ "$VERSION" != "latest" ]; then
  set -- "$@" --version "$VERSION"
fi
"$mach" "$@"

# --- best-effort environment check -----------------------------------------
"$mach" preflight || true
case ":${PATH}:" in
  *":$INSTALL_DIR:"*) : ;;
  *)
    if [ -z "${MACHINERY_BIN:-}" ]; then
      say ""
      say "note: $INSTALL_DIR is not on your PATH. Add it:"
      say "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    ;;
esac
