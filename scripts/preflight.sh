#!/usr/bin/env bash
# preflight.sh - run every required CI and formal gate locally, before a push.
#
# This mirrors the required .github workflows, including engine-backed formal,
# C4, and external-checker verification. Checks are ordered cheapest-first: a formatting slip fails in
# under a second instead of after the 20s race suite. Any failure exits non-zero.
#
# Bypass in an emergency with:  SKIP_PREFLIGHT=1 git push
# Run directly any time with:   make preflight   (or  scripts/preflight.sh)
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

if [ "${SKIP_PREFLIGHT:-0}" = "1" ]; then
  echo "preflight: SKIP_PREFLIGHT=1 set, skipping local gate suite" >&2
  exit 0
fi

preflight_work=$(mktemp -d)
cleanup() { rm -rf -- "$preflight_work"; }
trap cleanup EXIT
# shellcheck source=scripts/git-safe.sh
source "$script_dir/git-safe.sh"
git_safe_prepare "$repo_root" "$preflight_work/git-safe"

step=0
say()  { step=$((step + 1)); printf '\n\033[1m[preflight %d] %s\033[0m\n' "$step" "$1"; }
fail() { printf '\n\033[31mpreflight FAILED: %s\033[0m\n' "$1" >&2; exit 1; }

# A negative search has three outcomes: match, no match, or scan failure.
# Only grep status 1 is a clean result; unreadable/missing inputs must block.
reject_matches() {
  local finding=$1 scan_failure=$2 status=0
  shift 2
  "$@" || status=$?
  case "$status" in
    0) fail "$finding" ;;
    1) return 0 ;;
    *) fail "$scan_failure (status $status)" ;;
  esac
}

# 1. whitespace (ci: docs job) ---------------------------------------------
say "git diff --check (aggregate branch diff)"
trusted_base=${PREFLIGHT_BASE_REF:-origin/main}
if ! base=$(git_safe merge-base HEAD "$trusted_base"); then
  fail "cannot resolve trusted aggregate-diff base $trusted_base (fetch it or set PREFLIGHT_BASE_REF explicitly)"
fi
git_safe diff --check "$base" || fail "branch diff contains whitespace errors"

# 2. gofmt (ci: lint job, formatting gate) ---------------------------------
say "gofmt (formatting gate)"
unformatted=$(gofmt -l cmd/ internal/)
if [ -n "$unformatted" ]; then
  echo "$unformatted" >&2
  echo "fix with: gofmt -w cmd/ internal/" >&2
  fail "files are not gofmt-clean"
fi

# 3. go vet (ci: lint job) --------------------------------------------------
say "go vet ./..."
go vet ./... || fail "go vet reported problems"

# 4. golangci-lint (ci: lint job) ------------------------------------------
say "golangci-lint"
if command -v golangci-lint >/dev/null 2>&1; then
  want=$(cat .golangci-version 2>/dev/null)
  have=$(golangci-lint version --short 2>/dev/null)
  if [ -n "$want" ] && [ "${want#v}" != "${have#v}" ]; then
    fail "golangci-lint ${have:-unknown} does not match pin $want (run: make lint-install)"
  fi
  golangci-lint run --config .golangci.yml --timeout 5m || fail "golangci-lint reported problems"
else
  fail "golangci-lint is required at the version pinned in .golangci-version (run: make lint-install)"
fi

# 4b. GitHub Actions syntax/semantics (ci: lint job) -----------------------
say "actionlint"
if command -v actionlint >/dev/null 2>&1; then
  want=$(cat .actionlint-version 2>/dev/null)
  have=$(actionlint -version 2>/dev/null | sed -n '1p')
  if [ -n "$want" ] && [ "$want" != "$have" ]; then
    fail "actionlint ${have:-unknown} does not match pin $want (run: make lint-install)"
  fi
  actionlint .github/workflows/*.yml || fail "actionlint reported workflow problems"
else
  fail "actionlint is required at the version pinned in .actionlint-version (run: make lint-install)"
fi

# 4c. Shell script warnings (ci: lint job) ----------------------------------
say "shellcheck"
if command -v shellcheck >/dev/null 2>&1; then
  want=$(cat .shellcheck-version 2>/dev/null)
  have=$(shellcheck --version 2>/dev/null | awk '$1 == "version:" {print $2}')
  if [ -z "$want" ] || [ "$want" != "$have" ]; then
    fail "ShellCheck ${have:-unknown} does not match pin ${want:-missing}"
  fi
  shell_inventory=$(mktemp)
  if ! scripts/shellcheck-inventory.sh >"$shell_inventory"; then
    rm -f "$shell_inventory"
    fail "ShellCheck file inventory is incomplete or invalid"
  fi
  shell_files=()
  while IFS= read -r shell_file; do
    shell_files+=("$shell_file")
  done <"$shell_inventory"
  rm -f "$shell_inventory"
  [ "${#shell_files[@]}" -gt 0 ] || fail "ShellCheck file inventory is unexpectedly empty"
  shellcheck "${shell_files[@]}" || fail "shellcheck reported problems"
else
  fail "ShellCheck is required at the version pinned in .shellcheck-version"
fi

# 5. go.mod / go.sum tidy (ci: tidy job) -----------------------------------
say "go mod tidy (verify clean)"
go mod tidy || fail "go mod tidy errored"
if ! git_safe diff --quiet -- go.mod go.sum; then
  git_safe diff -- go.mod go.sum >&2 || true
  fail "go.mod/go.sum not tidy (the fix has been applied; review and stage it)"
fi

# 6. docs gate (ci: docs job) ----------------------------------------------
say "docs gate (no stale toolchain refs, no external-checker host-runtime drift, no em dashes)"
# The Python gate polices the machinery TOOLCHAIN surface (the same set ci.yml
# scans), not examples/: an external-checker example adapter is user-supplied and
# any-language by design (the pii-flow reference uses a digest-pinned OCI userspace),
# so examples/ is excluded here to match the authoritative CI gate.
reject_matches "stale Python-toolchain reference in the doc surface" \
  "stale Python-toolchain scan failed" \
  grep -rnE "PyYAML|pyyaml|uv run|oracle_gen\.py|machine_lint\.py|machinery_check\.py|tla_gen\.py|refine_gen\.py|compose_gen\.py|diff-all\.sh|capture-golden\.sh" \
  README.md CONTRIBUTING.md install.sh skills/ agents/ docs/ commands/ adapters/ hooks/ Makefile
reject_matches "stale host checker-runtime contract in the doc surface" \
  "host checker-runtime scan failed" \
  grep -rnEi "Souffl(e|é).*(external.checker|checker engine|CI pin|required)|external.checker.*Souffl(e|é)" \
  README.md docs/ examples/pii-flow/README.md scripts/preflight.sh .github/workflows/ci.yml
# Em dash encoded as UTF-8 octets because stock macOS Bash 3.2 does not decode
# \u escapes in ANSI-C strings.
em_dash=$(printf '\342\200\224')
reject_matches "em dash found in the doc surface (house style forbids it)" \
  "em-dash scan failed" \
  grep -rn "$em_dash" README.md CONTRIBUTING.md install.sh skills/ agents/ docs/ examples/ commands/ adapters/ hooks/ Makefile .github/

# 7. Modelith render freshness (ci: modelith-render job) -------------------
say "Modelith render freshness (pinned engine + mechanical house style)"
make modelith-render-check || fail "committed Modelith renders are stale or the pinned engine is unavailable"

# 8. build (ci: build + gates jobs) ----------------------------------------
say "build .bin/machinery"
make build || fail "build failed"

# 9. race tests (ci: test job) ---------------------------------------------
say "go test -race ./..."
go test -race -count=1 ./... || fail "unit/experiment tests failed"

# 10. golden corpus + adversarial experiments (ci: golden job) -------------
say "golden corpus + gate-experiment suite"
go test -count=1 -run TestGolden ./cmd/machinery || fail "golden corpus drifted (re-capture with: make golden-update)"
go test -count=1 ./internal/experiments/ || fail "adversarial gate-experiment suite failed"

# 11. example gate suites (ci: gates job) ----------------------------------
say "machinery check (all 8 example design suites)"
scripts/example-inventory.sh rows | while IFS=$'\t' read -r -a row; do
  design=${row[0]}
  impl=${row[1]}
  complete=${row[5]}
  args=("$design" --warnings-as-errors)
  [[ "$impl" == - ]] || args+=(--impl "$impl")
  [[ "$complete" == no ]] || args+=(--complete)
  .bin/machinery check "${args[@]}" || fail "gate suite: $design"
done

# 12. registered implementation module suites (ci: example-impls job) ------
say "registered implementation module tests"
scripts/example-inventory.sh impl-modules | while IFS=$'\t' read -r design impl module; do
  case "$module" in
    go) ( cd "$impl" && go test ./... -count=1 ) || fail "implementation tests failed: $design" ;;
    *) fail "unsupported implementation module type $module: $design" ;;
  esac
done

# 13. engine-backed formal suite (formal.yml) ------------------------------
say "formal verification (regeneration + TLC)"
make verify-formal || fail "formal verification failed"

# 14. C4 engine compilation (ci: engine-verification job) -----------------
say "C4 compilation (Structurizr CLI)"
STRUCTURIZR_VERSION=$(sed -n 's/^STRUCTURIZR_VERSION=//p' .structurizr-pin)
[ -n "$STRUCTURIZR_VERSION" ] || fail ".structurizr-pin does not declare STRUCTURIZR_VERSION"
# verify-c4 owns pinned, checksum-verified provisioning. Do not forward an
# ambient executable here: an override without its exact closure digest is
# intentionally rejected, and probing PATH would make preflight host-specific.
unset MACHINERY_STRUCTURIZR_CLI MACHINERY_STRUCTURIZR_CLI_CLOSURE_SHA256
c4_inventory=$preflight_work/c4.inventory
if ! scripts/c4-inventory.sh examples >"$c4_inventory"; then
  fail "C4 workspace discovery failed or returned an empty corpus"
fi
while IFS= read -r dsl; do
  .bin/machinery verify-c4 "$(dirname "$dsl")" || fail "C4 verification failed for $dsl"
done <"$c4_inventory"

# 15. external checker reproduction (ci: design-engines job) --------------
say "external checker reproduction (immutable OCI closure)"
docker_bin=$(command -v docker || true)
[ -n "$docker_bin" ] || fail "Docker is required for external-checker verification"
docker_bin=$(realpath "$docker_bin")
[ -f "$docker_bin" ] && [ -x "$docker_bin" ] || fail "Docker must resolve to a regular executable"
run_safe=$preflight_work/run-safe
run_safe_build_stderr=$preflight_work/run-safe-build.stderr
if ! go build -o "$run_safe" ./scripts/run-safe 2>"$run_safe_build_stderr"; then
  cat "$run_safe_build_stderr" >&2
  fail "could not build bounded external-command runner"
fi
if [ -s "$run_safe_build_stderr" ]; then
  cat "$run_safe_build_stderr" >&2
  fail "building bounded external-command runner emitted stderr; warnings are forbidden"
fi
checker_image=python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc
checker_platform=linux/amd64
pull_receipt=$("$run_safe" -timeout 10m -stdout-limit 4096 -stderr-limit 65536 -- \
  "$docker_bin" pull --quiet --platform "$checker_platform" "$checker_image") ||
  fail "could not provision pinned external-checker image $checker_image for $checker_platform"
case "$pull_receipt" in
  "$checker_image"|"docker.io/library/$checker_image") ;;
  *) fail "Docker pull returned a non-canonical image receipt: $pull_receipt" ;;
esac
repo_digests=$("$run_safe" -timeout 30s -stdout-limit 65536 -stderr-limit 4096 -- \
  "$docker_bin" image inspect --format '{{json .RepoDigests}} {{.Os}}/{{.Architecture}}' "$checker_image") ||
  fail "could not inspect pinned external-checker image"
case "$repo_digests" in
  *\"$checker_image\"*" $checker_platform") ;;
  *) fail "local OCI identity does not match $checker_image on $checker_platform: $repo_digests" ;;
esac
"$run_safe" -timeout 2m -stdout-limit 4096 -stderr-limit 4096 -- \
  "$docker_bin" run --rm --pull=never --platform "$checker_platform" --network=none --read-only \
  "$checker_image" python3 --version || fail "pinned external-checker image cannot run offline on $checker_platform"
if [ -z "${DOCKER_HOST:-}" ]; then
  DOCKER_HOST=$("$run_safe" -timeout 30s -stdout-limit 4096 -stderr-limit 4096 -- \
    "$docker_bin" context inspect --format '{{(index .Endpoints "docker").Host}}') ||
    fail "could not resolve the active Docker endpoint"
  case "$DOCKER_HOST" in
    ""|*$'\n'*|*[[:space:]]*) fail "Docker context returned a non-canonical endpoint" ;;
  esac
  export DOCKER_HOST
fi
checker_engine_dir=$(dirname "$docker_bin")
scripts/example-inventory.sh checkers | while IFS=$'\t' read -r design registry; do
  PATH="$checker_engine_dir:$PATH" .bin/machinery verify-checkers "$design" --registry "$registry" || fail "external checker verification failed: $design"
done

printf '\n\033[32mpreflight OK: all required local CI/formal gates passed.\033[0m\n'
