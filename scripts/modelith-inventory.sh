#!/usr/bin/env bash
# Discover the authoritative Modelith corpus at recipe execution time. Make's
# $(shell ...) discards command status, so discovery must live in a strict
# process where a failed find or sort cannot become a partial/empty success.
set -euo pipefail

action=${1:-check}
root=${2:-examples}
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)

sources=$(mktemp)
renders=$(mktemp)
entries=$(mktemp)
cleanup() {
  rm -f "$sources" "$renders" "$entries" "$entries.git-safe" "$entries.git-safe.stderr"
}
trap cleanup EXIT
# shellcheck source=scripts/git-safe.sh
source "$script_dir/git-safe.sh"

if [[ -L "$root" || ! -d "$root" ]]; then
  echo "Modelith corpus root must be a real directory: $root" >&2
  exit 1
fi
if ! find "$root" -print | LC_ALL=C sort >"$entries"; then
  echo "Modelith discovery failed during all-entry inventory; corpus is untrusted" >&2
  exit 1
fi
while IFS= read -r entry; do
  [[ "$entry" == "$root" ]] && continue
  if [[ -L "$entry" ]]; then
    echo "Modelith corpus must not contain symlinks: $entry" >&2
    exit 1
  fi
  if [[ ! -f "$entry" && ! -d "$entry" ]]; then
    echo "Modelith corpus entries must be regular files or real directories: $entry" >&2
    exit 1
  fi
done <"$entries"

if ! find "$root" -name '*.modelith.yaml' ! -path '*/pack/*' ! -path '*/packs/*' | LC_ALL=C sort >"$sources"; then
  echo "Modelith source discovery failed; inventory is untrusted" >&2
  exit 1
fi
if [[ ! -s "$sources" ]]; then
  echo "Modelith source discovery returned an unexpected empty corpus under $root" >&2
  exit 1
fi
if ! find "$root" -name '*.modelith.md' ! -path '*/pack/*' ! -path '*/packs/*' | LC_ALL=C sort >"$renders"; then
  echo "Modelith render discovery failed; inventory is untrusted" >&2
  exit 1
fi

check_pairs() {
  local source rendered
  while IFS= read -r source; do
    rendered=${source%.yaml}.md
    if [[ -L "$source" || ! -f "$source" ]]; then
      echo "Modelith source must be a regular non-symlink file: $source" >&2
      return 1
    fi
    if [[ -L "$rendered" || ! -f "$rendered" ]]; then
      echo "missing Modelith render for $source" >&2
      return 1
    fi
  done <"$sources"
  while IFS= read -r rendered; do
    source=${rendered%.md}.yaml
    if [[ -L "$rendered" || ! -f "$rendered" ]]; then
      echo "Modelith render must be a regular non-symlink file: $rendered" >&2
      return 1
    fi
    if [[ -L "$source" || ! -f "$source" ]]; then
      echo "orphan Modelith render without source: $rendered" >&2
      return 1
    fi
  done <"$renders"
}

case "$action" in
  sources)
    check_pairs
    cat "$sources"
    ;;
  renders)
    check_pairs
    cat "$renders"
    ;;
  check)
    check_pairs
    ;;
  git-diff)
    check_pairs
    git_safe_prepare "$repo_root" "$entries.git-safe"
    render_args=()
    while IFS= read -r rendered; do
      render_args+=("$rendered")
    done <"$renders"
    git_safe diff --exit-code HEAD -- "${render_args[@]}"
    status=$(git_safe status --porcelain=v1 --untracked-files=all -- "${render_args[@]}")
    if [[ -n "$status" ]]; then
      echo "Modelith render targets are not exactly committed:" >&2
      printf '%s\n' "$status" >&2
      exit 1
    fi
    ;;
  *)
    echo "usage: $0 {sources|renders|check|git-diff} [root]" >&2
    exit 2
    ;;
esac
