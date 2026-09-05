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
#   MACHINERY_HOMES        one agent home per line; the FIRST is canonical.
#                          A single path may contain spaces. For multiple paths,
#                          include literal newlines (for example with printf).
#                          Unset, the binary uses its defaults ("$HOME/.agents"
#                          then "$HOME/.claude") and skips any home the Claude
#                          Code plugin already serves; setting this passes the
#                          homes explicitly, which always wins over that skip.
#   MACHINERY_TARGETS      space-separated host adapters from claude, codex,
#                          opencode, all. Cannot be combined with MACHINERY_HOMES.
#   INSTALL_DIR            where the CLI binary lands (default: "$HOME/.local/bin")
#   MACHINERY_REPO         owner/name to fetch from (default: RamXX/machinery)
#   MACHINERY_REQUIRE_PREFLIGHT
#                          set to 1 to fail the script when the closing prerequisite
#                          check finds a gap (default: report the gap, exit 0; the
#                          exit status reports the install, not the toolchain)
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
case "$REPO" in
  */*) ;;
  *) die "MACHINERY_REPO must be owner/name" ;;
esac
case "$VERSION" in
  latest) ;;
  v[0-9]*.[0-9]*.[0-9]*)
	printf '%s\n' "$VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?(\+[0-9A-Za-z][0-9A-Za-z.-]*)?$' ||
	  die "invalid MACHINERY_VERSION: $VERSION (want latest or an exact vMAJOR.MINOR.PATCH tag)"
	;;
  *) die "invalid MACHINERY_VERSION: $VERSION (want latest or an exact vMAJOR.MINOR.PATCH tag)" ;;
esac

# --- detect os/arch (must match the release asset names) -------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported architecture: $arch" ;;
esac
case "$os" in
  linux|darwin) ;;
  msys*|mingw*|cygwin*|windows*) die "Windows is not a supported binary release target in v0.6.11" ;;
  *) die "unsupported OS: $os" ;;
esac
binname="machinery"

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
	candidate=""
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
	printf '%s\n' "$TAG" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?(\+[0-9A-Za-z][0-9A-Za-z.-]*)?$' ||
	  die "release API returned an invalid tag: $TAG"
  say "machinery $TAG ($os/$arch)"
  asset="machinery-${os}-${arch}"
  base="https://github.com/$REPO/releases/download/$TAG"
  say "Downloading $asset..."
  curl -fsSL -o "$tmp/$asset" "$base/$asset" || die "failed to download $asset from $TAG"
  curl -fsSL -o "$tmp/checksums-sha256.txt" "$base/checksums-sha256.txt" 2>/dev/null \
    || die "release $TAG has no checksums-sha256.txt; refusing to install an unverified binary"
  want=$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums-sha256.txt")
  got=$(sha256 "$tmp/$asset")
  [ -n "$want" ] || die "no checksum listed for $asset"
	case "$want" in
	  *' '*|*'
'*) die "duplicate or malformed checksum entries for $asset" ;;
	esac
	printf '%s\n' "$want" | grep -Eq '^[0-9a-fA-F]{64}$' || die "invalid checksum listed for $asset"
  [ "$want" = "$got" ] || die "checksum mismatch for $asset (want $want, got $got)"
  say "checksum verified"
  mkdir -p "$INSTALL_DIR"
	candidate="$tmp/$binname.candidate"
	cp "$tmp/$asset" "$candidate"
	chmod +x "$candidate"
	reported=$("$candidate" version 2>&1) || die "downloaded binary failed its version probe: $reported"
	[ "$reported" = "machinery version $TAG" ] || die "downloaded binary reports '$reported', want 'machinery version $TAG'"
	mach="$candidate"
fi

# --- place the skill + role docs (the binary owns this) --------------------
# No --home flags unless MACHINERY_HOMES is set: the binary's default home
# list already skips any home the Claude Code plugin serves, and an explicit
# --home is the documented way to override that.
if [ -n "${MACHINERY_BIN:-}" ]; then
	set -- install
else
	# The verified candidate delegates the complete binary + harness + receipt
	# mutation to machinery's locked, crash-recoverable update transaction. The
	# bootstrap script never opens its own missing-target replacement window.
	set -- update --version "$TAG" --repo "$REPO" --install-dir "$INSTALL_DIR" --skip-plugins
	if [ -z "$HOMES" ] && [ -z "$TARGETS" ]; then
		set -- "$@" --bootstrap-defaults
	fi
fi
while IFS= read -r h; do
	[ -n "$h" ] || continue
  set -- "$@" --home "$h"
done <<EOF
$HOMES
EOF
target_lines=$(printf '%s\n' "$TARGETS" | tr '[:space:]' '\n')
while IFS= read -r target; do
	[ -n "$target" ] || continue
  set -- "$@" --target "$target"
done <<EOF
$target_lines
EOF
if [ -n "${MACHINERY_SKILL_SRC:-}" ]; then
	[ -n "${MACHINERY_BIN:-}" ] || die "MACHINERY_SKILL_SRC requires MACHINERY_BIN for an offline bootstrap"
  set -- "$@" --from "$MACHINERY_SKILL_SRC"
fi
if [ -n "${MACHINERY_BIN:-}" ] && [ "$VERSION" != "latest" ]; then
  set -- "$@" --version "$VERSION"
fi
"$mach" "$@"
if [ -z "${MACHINERY_BIN:-}" ]; then
	mach="$INSTALL_DIR/$binname"
	say "installed $binname -> $mach"
fi

# --- environment check -----------------------------------------------------
# The install is complete at this point, so the exit status of this script
# reports the install. The prerequisite check is advisory: a missing tool
# (typically modelith, which only Phase 1 authoring needs; `machinery check`
# does not) is printed, not fatal. MACHINERY_REQUIRE_PREFLIGHT=1 makes it fatal
# for images that must ship the complete toolchain.
if ! "$mach" preflight; then
  if [ "${MACHINERY_REQUIRE_PREFLIGHT:-0}" = "1" ]; then
    die "prerequisite check failed and MACHINERY_REQUIRE_PREFLIGHT=1 is set (the install itself succeeded)"
  fi
  say ""
  say "note: the install succeeded; the prerequisite check above found gaps. Install the missing tools, then run: machinery preflight"
fi
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
