#!/usr/bin/env bash
# preflight.sh - run every required CI and formal gate locally, in one go.
#
# This is the full mirror of the required .github workflows, including
# engine-backed formal, C4, and external-checker verification. It is a
# deliberate local run (`make preflight`), not the push gate: pushing runs the
# cheap tier only, and hosted CI is the authority for the heavy tier.
#
# Two tiers, one owner each:
#   scripts/preflight-fast.sh   cheap tier, also what .githooks/pre-push runs
#   this script                 the fast tier, then the heavy tier below
#
# Heavy tier: the race sweep, the required native integration lane, the
# registered implementation modules, TLC formal verification, C4 compilation,
# and external checker reproduction. Any failure exits non-zero.
#
# Run directly any time with:   make preflight   (or  scripts/preflight.sh)
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

preflight_work=$(mktemp -d)
cleanup() { rm -rf -- "$preflight_work"; }
trap cleanup EXIT

step=0
say()  { step=$((step + 1)); printf '\n\033[1m[preflight %d] %s\033[0m\n' "$step" "$1"; }
fail() { printf '\n\033[31mpreflight FAILED: %s\033[0m\n' "$1" >&2; exit 1; }

# 1. cheap tier (single owner, shared verbatim with the pre-push hook) ------
say "fast tier (scripts/preflight-fast.sh)"
scripts/preflight-fast.sh || fail "cheap gate tier failed"

# 2. native assurance runtimes (presence only) -----------------------------
# The required lane's four-language catalog verifies exact identities and
# fails closed; this early check only avoids burning minutes before the
# missing prerequisite is reported there.
say "native assurance runtime presence (node, python3, elixir, mix, tsc)"
for tool in node python3 elixir mix tsc; do
  command -v "$tool" >/dev/null 2>&1 ||
    fail "required native assurance runtime '$tool' is missing (the lane pins Node 26.9.0 + TypeScript 7.0.2, CPython 3.14.7, Elixir/Mix 1.20.4 with OTP 29 / ERTS 17.1)"
done

# 3. race tests (ci: test job) ---------------------------------------------
say "go test -race ./..."
# The per-binary alarm must sit above the slowest legitimate package:
# internal/install's serial fsync-bound suite measures ~1193s under race on
# the documented host class (evidence run), leaving the previous 20m alarm
# 0.5% of margin; ordinary IO jitter crossed it mid-fsync. 30m still kills
# any hung test fail-closed while covering the realistic cost, and stays
# below the mirrored CI job's own 40m budget.
go test -race -count=1 ./... -timeout=30m || fail "unit/experiment tests failed"

# Runtime tests carry machinery_integration and are absent from the native
# suite above. Provision the pinned closure before selecting them here.
say "required native integration lane (Docker, pinned Java/TLC, Node)"
go run ./scripts/integration-lane --lane required

# 4. registered implementation module suites (ci: example-impls job) -------
say "registered implementation module tests"
scripts/example-inventory.sh impl-modules | while IFS=$'\t' read -r design impl module; do
  case "$module" in
    go) ( cd "$impl" && go test ./... -count=1 ) || fail "implementation tests failed: $design" ;;
    *) fail "unsupported implementation module type $module: $design" ;;
  esac
done

# 5. engine-backed formal suite (formal.yml) -------------------------------
say "formal verification (regeneration + TLC)"
make verify-formal || fail "formal verification failed"

# 6. C4 engine compilation (ci: engine-verification job) -------------------
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

# 7. external checker reproduction (ci: design-engines job) ----------------
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
