#!/usr/bin/env bash
# ci-linux.sh - reproduce the two hosted Linux evidence jobs locally.
#
# Runs, inside a pinned 2-CPU linux/amd64 container that carries the exact
# runtime identities .github/actions/assurance-runtimes installs:
#
#   ci.yml test job                  go test -race -count=1 ./... -timeout=30m
#   ci.yml integration-required job  go run ./scripts/integration-lane --lane required
#
# This is the step before any push that touches custody, adapters, or the
# lane: those are the failures a macOS-only gate cannot see (2-vCPU timing,
# Linux-only build tags, a cold module cache).
#
# What it does reproduce: the Linux kernel and filesystem semantics, the
# 2-CPU scheduling pressure the hosted runner applies, the linux/amd64 build
# of every package, and the lane's pinned runtime identities.
#
# What it does not reproduce: hosted-runner hardware and disk speed (absolute
# wall times differ), the hosted image's preinstalled tool surface beyond the
# pinned runtimes, and the macOS-only jobs (native-tests, golden-native).
# The lane's Docker work runs against the HOST daemon through the mounted
# socket, not a nested daemon, so a host daemon on a different architecture
# emulates the pinned linux/amd64 checker image exactly as it does outside
# the container.
#
# On an arm64 host, an emulated linux/amd64 race sweep is impractically slow;
# run this on a linux/amd64 machine or VM for evidence. Under emulation the
# BEAM's JIT also traps, so an exploratory emulated run needs
# ERL_FLAGS="+JMsingle true" in the environment, which this script forwards.
#
# Run directly any time with:   make ci-linux   (or  scripts/ci-linux.sh)
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

platform=${MACHINERY_CI_LINUX_PLATFORM:-linux/amd64}
cpus=${MACHINERY_CI_LINUX_CPUS:-2}
image=${MACHINERY_CI_LINUX_IMAGE:-machinery-ci-linux:local}
socket=${MACHINERY_CI_LINUX_DOCKER_SOCKET:-/var/run/docker.sock}
# A CPU quota alone still shows every host core to the guest, and a runtime
# that sizes its scheduler from the visible core count (the BEAM, the Go
# runtime, the test binaries) would then behave nothing like a 2-vCPU runner.
# Pin the cpuset too, so the count the guest sees is the count it gets.
cpuset=${MACHINERY_CI_LINUX_CPUSET:-0-$((cpus - 1))}

fail() { printf '\n\033[31mci-linux FAILED: %s\033[0m\n' "$1" >&2; exit 1; }
say()  { printf '\n\033[1m[ci-linux] %s\033[0m\n' "$1"; }

command -v docker >/dev/null 2>&1 || fail "Docker is required (the lane provisions a pinned OCI closure)"
docker version >/dev/null 2>&1 || fail "the Docker daemon is not reachable"
[ -S "$socket" ] || fail "Docker socket $socket is not a socket (set MACHINERY_CI_LINUX_DOCKER_SOCKET)"

# Each stage of the image is repinnable without editing the Dockerfile, so a
# moved upstream tag is a one-line override rather than a patch.
build_args=()
repin() {
  if [ -n "$2" ]; then
    build_args+=(--build-arg "$1_IMAGE=$2")
  fi
  return 0
}
repin GO "${MACHINERY_CI_LINUX_GO_IMAGE:-}"
repin NODE "${MACHINERY_CI_LINUX_NODE_IMAGE:-}"
repin PYTHON "${MACHINERY_CI_LINUX_PYTHON_IMAGE:-}"
repin ELIXIR "${MACHINERY_CI_LINUX_ELIXIR_IMAGE:-}"
repin DOCKER_CLI "${MACHINERY_CI_LINUX_DOCKER_CLI_IMAGE:-}"

say "building $image for $platform (pinned Go, Node, TypeScript, CPython, Elixir/OTP)"
docker build --platform "$platform" -f scripts/ci-linux.dockerfile -t "$image" \
  ${build_args[@]+"${build_args[@]}"} . ||
  fail "image build failed (repin a moved stage with MACHINERY_CI_LINUX_<STAGE>_IMAGE; see scripts/ci-linux.dockerfile)"

evidence=$(mktemp -d)
say "evidence directory: $evidence"

# Caches live in named volumes: the run must not leave container-owned build
# artifacts in the worktree, and a cold module cache is the slowest part of a
# repeat run.
docker volume create machinery-ci-linux-gocache >/dev/null
docker volume create machinery-ci-linux-gomodcache >/dev/null

say "hosted ci test job + integration-required job on $cpus CPUs"
docker run --rm --platform "$platform" --cpus "$cpus" --cpuset-cpus "$cpuset" \
  -e GOMAXPROCS="$cpus" \
  ${ERL_FLAGS:+-e ERL_FLAGS="$ERL_FLAGS"} \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomodcache \
  -e MACHINERY_INTEGRATION_REPORT_DIR=/evidence \
  -v "$repo_root:/src" \
  -v "$evidence:/evidence" \
  -v machinery-ci-linux-gocache:/gocache \
  -v machinery-ci-linux-gomodcache:/gomodcache \
  -v "$socket:/var/run/docker.sock" \
  -w /src "$image" \
  bash -euo pipefail -c '
    go version
    node --version
    python3 --version
    elixir --version
    tsc --version
    go test -race -count=1 ./... -timeout=30m
    go run ./scripts/integration-lane --lane required
  ' || fail "the containerized race sweep or required lane failed (evidence: $evidence)"

printf '\n\033[32mci-linux OK: race sweep and required lane reproduced on %s with %s CPUs.\033[0m\n' "$platform" "$cpus"
printf 'lane evidence retained in %s\n' "$evidence"
