#!/usr/bin/env bash
# Run the pinned Modelith renderer as a closed, crash-recoverable repository
# transformation: exact tool output, no stderr, and one atomic corpus publish.
set -euo pipefail

action=${1:-render}
corpus=${2:-examples}
pin=${3:-v0.4.0}
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

case "$action" in
  render) ;;
  *)
    echo "usage: $0 render [examples] [pinned-version]" >&2
    exit 2
    ;;
esac
if [[ "$corpus" != "examples" ]]; then
  echo "Modelith transaction supports only the authoritative examples corpus" >&2
  exit 2
fi
if [[ ! "$pin" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid pinned Modelith version: $pin" >&2
  exit 2
fi
modelith_source=$(command -v modelith || true)
[[ -n "$modelith_source" ]] || {
  echo "modelith $pin is required (go install github.com/stacklok/modelith/cmd/modelith@$pin)" >&2
  exit 1
}

work=$(mktemp -d)
transaction=$work/modelith-tx
run_safe=$work/run-safe
tree_inventory=$work/tree-inventory
modelith_bin=$work/modelith
modelith_receipt=$work/modelith.receipt
stage_root=$repo_root/.machinery-modelith-stage
cleanup() {
  local status=$?
  if [[ -x "$transaction" ]]; then
    "$transaction" recover "$repo_root" >/dev/null 2>&1 || true
  fi
  rm -rf -- "$work"
  return "$status"
}
trap cleanup EXIT
# shellcheck source=scripts/git-safe.sh
source "$script_dir/git-safe.sh"
git_safe_prepare "$repo_root" "$work/git-safe"

go_build_stderr=$work/go-build.stderr
go_mod_stdout=$work/go-mod.stdout
if ! go mod download >"$go_mod_stdout" 2>"$go_build_stderr"; then
  echo "download pinned Go module closure failed" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if [[ -s "$go_mod_stdout" || -s "$go_build_stderr" ]]; then
  echo "downloading pinned Go module closure emitted output; warnings are forbidden" >&2
  cat "$go_mod_stdout" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if ! go build -o "$transaction" ./scripts/modelith-tx.go 2>"$go_build_stderr"; then
  echo "build Modelith transaction helper failed" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if [[ -s "$go_build_stderr" ]]; then
  echo "building Modelith transaction helper emitted stderr; warnings are forbidden" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if ! go build -o "$run_safe" ./scripts/run-safe 2>"$go_build_stderr"; then
  echo "build bounded external-command helper failed" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if [[ -s "$go_build_stderr" ]]; then
  echo "building bounded external-command helper emitted stderr; warnings are forbidden" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if ! go build -o "$tree_inventory" ./scripts/tree-inventory 2>"$go_build_stderr"; then
  echo "build bounded tree inventory helper failed" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if [[ -s "$go_build_stderr" ]]; then
  echo "building bounded tree inventory helper emitted stderr; warnings are forbidden" >&2
  cat "$go_build_stderr" >&2
  exit 1
fi
if ! "$run_safe" snapshot-executable -source "$modelith_source" -destination "$modelith_bin" -receipt "$modelith_receipt"; then
  echo "snapshot pinned Modelith executable failed" >&2
  exit 1
fi
"$transaction" recover "$repo_root"

sources=$work/sources
renders=$work/renders
render_excludes=$work/render-excludes
"$script_dir/modelith-inventory.sh" sources "$corpus" >"$sources"
: >"$renders"
: >"$render_excludes"
while IFS= read -r source; do
  rendered=${source%.yaml}.md
  printf '%s\n' "$rendered" >>"$renders"
  printf '%s\n' "${rendered#"$corpus"/}" >>"$render_excludes"
done <"$sources"

snapshot_repo() {
  local destination=$1
  if ! "$tree_inventory" -snapshot -root . -prune .git -exclude-file "$renders" \
    -max-entries 200000 -max-depth 128 -max-bytes 67108864 \
    -max-file-bytes 33554432 -max-total-bytes 134217728 -timeout 30s >"$destination"; then
    echo "bounded repository snapshot failed" >&2
    return 1
  fi
}

snapshot_corpus_without_renders() {
  local root=$1 destination=$2
  if ! "$tree_inventory" -snapshot -root "$root" -exclude-file "$render_excludes" \
    -max-entries 100000 -max-depth 64 -max-bytes 33554432 \
    -max-file-bytes 8388608 -max-total-bytes 33554432 -timeout 15s >"$destination"; then
    echo "bounded Modelith corpus snapshot failed" >&2
    return 1
  fi
}

snapshot_status() {
  local destination=$1 rendered
  local -a status_args=(-- .)
  while IFS= read -r rendered; do
    status_args+=(":(exclude)$rendered")
  done <"$renders"
  git_safe status --porcelain=v1 --untracked-files=all "${status_args[@]}" >"$destination"
}

before_inventory=$work/before.inventory
after_inventory=$work/after.inventory
before_status=$work/before.status
after_status=$work/after.status
before_corpus=$work/before.corpus
staged_corpus=$work/staged.corpus
snapshot_repo "$before_inventory"
snapshot_status "$before_status"
snapshot_corpus_without_renders "$corpus" "$before_corpus"
expected_digest=$("$transaction" fingerprint "$repo_root/$corpus")

version_stdout=$work/version.stdout
version_stderr=$work/version.stderr
version_expected=$work/version.expected
version_expected_prefixed=$work/version.expected-prefixed
printf 'modelith version %s\n' "${pin#v}" >"$version_expected"
printf 'modelith version %s\n' "$pin" >"$version_expected_prefixed"
if ! "$run_safe" -timeout 10s -stdout-limit 4096 -stderr-limit 4096 \
  -executable-receipt "$modelith_receipt" \
  -- "$modelith_bin" --version >"$version_stdout" 2>"$version_stderr"; then
  echo "modelith --version failed" >&2
  cat "$version_stdout" >&2
  cat "$version_stderr" >&2
  exit 1
fi
if [[ -s "$version_stderr" ]]; then
  echo "modelith --version emitted stderr; warnings are forbidden" >&2
  cat "$version_stderr" >&2
  exit 1
fi
if ! cmp -s "$version_expected" "$version_stdout" && \
  ! cmp -s "$version_expected_prefixed" "$version_stdout"; then
  echo "modelith version output does not exactly match pin $pin" >&2
  diff -u "$version_expected" "$version_stdout" >&2 || true
  exit 1
fi

if ! mkdir -m 0700 -- "$stage_root"; then
  echo "create private Modelith stage failed" >&2
  exit 1
fi
if ! cp -R -p -- "$corpus" "$stage_root/$corpus"; then
  echo "copy Modelith corpus to private stage failed" >&2
  exit 1
fi

em_dash=$(printf '\342\200\224')
while IFS= read -r source; do
  rendered=${source%.yaml}.md
  staged_source=$stage_root/$source
  staged_rendered=$stage_root/$rendered
  if [[ -L "$staged_source" || ! -f "$staged_source" ]]; then
    echo "Modelith source must remain a regular non-symlink file: $source" >&2
    exit 1
  fi
  if [[ -L "$staged_rendered" || ! -f "$staged_rendered" ]]; then
    echo "Modelith render target must be an existing regular non-symlink file: $rendered" >&2
    exit 1
  fi
  : >"$work/render.stdout"
  : >"$work/render.stderr"
  : >"$work/render.expected-stdout"
  printf 'wrote %s\n' "$rendered" >"$work/render.expected-stderr"
  if ! (cd "$stage_root" && "$run_safe" -timeout 2m -stdout-limit 4096 -stderr-limit 4096 \
    -expect-stdout-file "$work/render.expected-stdout" -expect-stderr-file "$work/render.expected-stderr" \
    -executable-receipt "$modelith_receipt" -- "$modelith_bin" render "$source") \
    >"$work/render.stdout" 2>"$work/render.stderr"; then
    echo "modelith render failed for $source" >&2
    cat "$work/render.stdout" >&2
    cat "$work/render.stderr" >&2
    exit 1
  fi
  if [[ -s "$work/render.stdout" ]]; then
    echo "modelith render emitted unexpected stdout for $source" >&2
    cat "$work/render.stdout" >&2
    exit 1
  fi
  if ! cmp -s "$work/render.expected-stderr" "$work/render.stderr"; then
    echo "modelith render emitted stderr that is not the canonical success receipt for $source; warnings are forbidden" >&2
    cat "$work/render.stderr" >&2
    exit 1
  fi
  if [[ -L "$staged_rendered" || ! -f "$staged_rendered" ]]; then
    echo "modelith replaced $rendered with a symlink or special entry" >&2
    exit 1
  fi
  LC_ALL=C sed "s/$em_dash/-/g" "$staged_rendered" |
    LC_ALL=C awk '{ line[NR] = $0 } END { n = NR; while (n > 0 && line[n] == "") n--; for (i = 1; i <= n; i++) print line[i] }' >"$work/normalized"
  cat "$work/normalized" >"$staged_rendered"
done <"$sources"

"$script_dir/modelith-inventory.sh" check "$stage_root/$corpus"
snapshot_corpus_without_renders "$stage_root/$corpus" "$staged_corpus"
if ! cmp -s "$before_corpus" "$staged_corpus"; then
  echo "modelith render changed staged corpus paths outside the exact render inventory" >&2
  diff -u "$before_corpus" "$staged_corpus" >&2 || true
  exit 1
fi
if ! "$run_safe" verify-executable -receipt "$modelith_receipt"; then
  echo "Modelith executable source or private snapshot changed before publication" >&2
  exit 1
fi
"$transaction" publish "$repo_root" "$expected_digest"
if ! "$run_safe" verify-executable -receipt "$modelith_receipt"; then
  echo "Modelith executable source or private snapshot changed during publication" >&2
  exit 1
fi

"$script_dir/modelith-inventory.sh" check "$corpus"
snapshot_repo "$after_inventory"
snapshot_status "$after_status"
if ! cmp -s "$before_inventory" "$after_inventory"; then
  echo "modelith render changed repository paths outside the exact render inventory" >&2
  diff -u "$before_inventory" "$after_inventory" >&2 || true
  exit 1
fi
if ! cmp -s "$before_status" "$after_status"; then
  echo "modelith render changed Git status outside the exact render inventory" >&2
  diff -u "$before_status" "$after_status" >&2 || true
  exit 1
fi
