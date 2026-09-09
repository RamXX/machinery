#!/usr/bin/env bash
# preflight-fast.sh - the cheap gate tier, the one the pre-push hook runs.
#
# One owner for the cheap tier. `.githooks/pre-push` execs this script and
# nothing else, and `scripts/preflight.sh` runs it as its first phase before
# the heavy tier, so the two cannot drift apart. Everything here is static
# analysis, a build, or a deterministic corpus: no race sweep, no runtime
# lane, no engine provisioning. Budget: under 5 minutes on the documented
# host class.
#
# The heavy tier (race sweep, required native lane, formal/TLC, C4, external
# checkers, the native macOS suite) is authoritative in hosted CI. Run it
# locally on purpose with `make preflight`, or on Linux with `make ci-linux`.
#
# There is no bypass variable here or in the hook.
#
# Run directly any time with:   make preflight-fast   (or  scripts/preflight-fast.sh)
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

preflight_work=$(mktemp -d)
cleanup() { rm -rf -- "$preflight_work"; }
trap cleanup EXIT
# shellcheck source=scripts/git-safe.sh
source "$script_dir/git-safe.sh"
git_safe_prepare "$repo_root" "$preflight_work/git-safe"

step=0
say()  { step=$((step + 1)); printf '\n\033[1m[fast %d] %s\033[0m\n' "$step" "$1"; }
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
# Run twice: once for the host platform, once for GOOS=linux. The hosted lint
# job is Linux, so a build-tagged or platform-specific file that only compiles
# on darwin is invisible to a macOS-only run and fails in CI instead. The
# second pass costs a warm re-analysis, not a second cold one.
say "golangci-lint (host and GOOS=linux)"
if command -v golangci-lint >/dev/null 2>&1; then
  want=$(cat .golangci-version 2>/dev/null)
  have=$(golangci-lint version --short 2>/dev/null)
  if [ -n "$want" ] && [ "${want#v}" != "${have#v}" ]; then
    fail "golangci-lint ${have:-unknown} does not match pin $want (run: make lint-install)"
  fi
  golangci-lint run --config .golangci.yml --timeout 5m || fail "golangci-lint reported problems"
  GOOS=linux golangci-lint run --config .golangci.yml --timeout 5m ||
    fail "golangci-lint reported problems for GOOS=linux (the platform hosted CI lints)"
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
  README.md docs/ examples/pii-flow/README.md scripts/preflight.sh scripts/preflight-fast.sh .github/workflows/ci.yml
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

# 9. example gate suites (ci: gates job) -----------------------------------
say "machinery check (all 8 example design suites)"
# The accepted two-policy gate is owned by one script, shared verbatim with
# the CI gates job and the Makefile check target so the three mirrors cannot
# drift apart. Read scripts/example-gates.sh for the policy itself.
scripts/example-gates.sh .bin/machinery || fail "example gate suites failed"

# 10. golden corpus + adversarial experiments (ci: golden job) -------------
say "golden corpus + gate-experiment suite"
go test -count=1 -run TestGolden ./cmd/machinery || fail "golden corpus drifted (re-capture with: make golden-update)"
go test -count=1 ./internal/experiments/ || fail "adversarial gate-experiment suite failed"

printf '\n\033[32mpreflight fast tier OK: every cheap gate passed. Heavy evidence: hosted CI, make preflight, make ci-linux.\033[0m\n'
