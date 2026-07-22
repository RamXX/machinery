#!/usr/bin/env bash
# preflight.sh - run the ci.yml gate suite locally, before a push.
#
# This mirrors .github/workflows/ci.yml job-for-job so a clean run here means a
# green run there. Checks are ordered cheapest-first: a formatting slip fails in
# under a second instead of after the 20s race suite. Any failure exits non-zero.
#
# Bypass in an emergency with:  SKIP_PREFLIGHT=1 git push
# Run directly any time with:   make preflight   (or  scripts/preflight.sh)
set -uo pipefail

cd "$(git rev-parse --show-toplevel)"

if [ "${SKIP_PREFLIGHT:-0}" = "1" ]; then
  echo "preflight: SKIP_PREFLIGHT=1 set, skipping local gate suite" >&2
  exit 0
fi

step=0
say()  { step=$((step + 1)); printf '\n\033[1m[preflight %d] %s\033[0m\n' "$step" "$1"; }
fail() { printf '\n\033[31mpreflight FAILED: %s\033[0m\n' "$1" >&2; exit 1; }

# 1. whitespace (ci: docs job) ---------------------------------------------
say "git diff --check (aggregate branch diff)"
base=$(git merge-base HEAD origin/main 2>/dev/null || git rev-parse HEAD^)
git diff --check "$base" || fail "branch diff contains whitespace errors"

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
    printf '\033[33m  golangci-lint %s installed but .golangci-version pins %s.\033[0m\n' "${have:-unknown}" "$want" >&2
    printf '\033[33m  CI runs %s; match it with: make lint-install\033[0m\n' "$want" >&2
  fi
  golangci-lint run --config .golangci.yml --timeout 5m || fail "golangci-lint reported problems"
else
  printf '\033[33m  golangci-lint not installed; skipping (CI still enforces it).\033[0m\n' >&2
  printf '\033[33m  install the pinned version with: make lint-install\033[0m\n' >&2
fi

# 5. go.mod / go.sum tidy (ci: tidy job) -----------------------------------
say "go mod tidy (verify clean)"
go mod tidy || fail "go mod tidy errored"
if ! git diff --quiet -- go.mod go.sum; then
  git diff -- go.mod go.sum >&2
  fail "go.mod/go.sum not tidy (the fix has been applied; review and stage it)"
fi

# 6. docs gate (ci: docs job) ----------------------------------------------
say "docs gate (no stale Python refs, no em dashes)"
if grep -rnE "python3|PyYAML|pyyaml|uv run|oracle_gen\.py|machine_lint\.py|machinery_check\.py|tla_gen\.py|refine_gen\.py|compose_gen\.py|diff-all\.sh|capture-golden\.sh" \
    README.md CONTRIBUTING.md install.sh skills/ agents/ docs/ examples/ commands/ adapters/ hooks/ Makefile; then
  fail "stale Python-toolchain reference in the doc surface"
fi
# Em dash spelled as an ANSI-C escape so this script stays free of the literal.
if grep -rn $'\u2014' README.md CONTRIBUTING.md install.sh skills/ agents/ docs/ examples/ commands/ adapters/ hooks/ Makefile .github/; then
  fail "em dash found in the doc surface (house style forbids it)"
fi

# 7. build (ci: build + gates jobs) ----------------------------------------
say "build .bin/machinery"
make build || fail "build failed"

# 8. race tests (ci: test job) ---------------------------------------------
say "go test -race ./..."
go test -race -count=1 ./... || fail "unit/experiment tests failed"

# 9. golden corpus + adversarial experiments (ci: golden job) --------------
say "golden corpus + gate-experiment suite"
go test -count=1 -run TestGolden ./cmd/machinery || fail "golden corpus drifted (re-capture with: make golden-update)"
go test -count=1 ./internal/experiments/ || fail "adversarial gate-experiment suite failed"

# 10. example gate suites (ci: gates job) ----------------------------------
say "machinery check (all 7 example design suites)"
.bin/machinery check examples/go-crm/design --impl examples/go-crm/impl || fail "gate suite: go-crm"
.bin/machinery check examples/surreal-crm/design                       || fail "gate suite: surreal-crm"
.bin/machinery check examples/fulfillment/design                       || fail "gate suite: fulfillment"
.bin/machinery check examples/portfolio-engine/design                  || fail "gate suite: portfolio-engine"
.bin/machinery check examples/checkout-split/parent/design             || fail "gate suite: checkout-split/parent"
.bin/machinery check examples/checkout-split/orders/design             || fail "gate suite: checkout-split/orders"
.bin/machinery check examples/checkout-split/payments/design           || fail "gate suite: checkout-split/payments"

# 11. go-crm impl hermetic suite (ci: go-crm-impl job) ---------------------
say "go-crm impl tests (separate module)"
( cd examples/go-crm/impl && go test ./... -count=1 ) || fail "go-crm impl tests failed"

printf '\n\033[32mpreflight OK: local gates match ci.yml, safe to push.\033[0m\n'
