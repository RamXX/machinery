# Contributing to machinery

This file is for people hacking on machinery itself (the Go source tree). End
users never need a clone, Make, or this file: they install the binary and run
`machinery` subcommands on their own designs.

## One-time setup

After cloning, arm the local gates once:

```sh
make hooks         # install the git pre-push hook (sets core.hooksPath)
make lint-install  # install the golangci-lint version pinned in .golangci-version
```

`make hooks` is required per clone. `core.hooksPath` is local git config, not
something the repo can commit for you, so a fresh checkout has no hook until you
run it. The hook and the script it runs are committed and shared; only the
one-line wiring is per-machine.

## The gate, in two tiers

The gate is tiered by cost. Every check has exactly one owner; the tiers differ
only in which checks they run.

**Cheap tier (`scripts/preflight-fast.sh`), on every push.** `make hooks` points
git at `.githooks/pre-push`, which runs this script and nothing else: the
aggregate whitespace diff, gofmt, vet, golangci-lint for the host and for
`GOOS=linux`, actionlint, ShellCheck, `go mod tidy`, the docs gate, the Modelith
render check, the build, the example gate suites, and the golden corpus plus the
gate-experiment suite. Budget: under 5 minutes. A formatting slip, a Linux-only
lint failure, or a golden drift fails here instead of after a red CI run.

```sh
make preflight-fast   # the same tier, on demand
```

**Heavy tier, authoritative in hosted CI.** The race sweep, the required native
integration lane, formal verification with TLC, C4 compilation, the external
checker reproduction, and the native macOS suite run in
`.github/workflows/ci.yml` and `formal.yml` on the pushed commit. Nothing local
is authoritative for them. Run the full mirror locally when you want it:

```sh
make preflight        # the cheap tier, then the heavy tier
```

(Not to be confused with `machinery preflight`, the end-user subcommand that
checks toolchain prerequisites. `make preflight` is the contributor CI mirror.)

There is no bypass variable in either tier: the hook runs on every push, and
hosted CI re-runs every gate on the pushed commit. Release publication is gated
separately on the exact commit carrying successful ci, formal, and security
runs (see `docs/release-policy.json`), so a red push can never publish.

## Linux evidence before a custody, adapter, or lane push

Run this before any push that touches custody (`internal/processscope`,
`internal/processcontrol`, `internal/install`), the TDD adapters
(`internal/tdd/adapters`), or the integration lane itself:

```sh
make ci-linux
```

It runs the hosted `test` job (`go test -race -count=1 ./... -timeout=30m`) and
the hosted `integration-required` job (`go run ./scripts/integration-lane --lane
required`) inside a pinned 2-CPU `linux/amd64` container carrying the exact
runtime identities `.github/actions/assurance-runtimes` installs: Go 1.27.1,
Node 26.8.1 with TypeScript 7.0.2, CPython 3.14.7, and Elixir 1.20.4 on OTP
29.0.6. The lane re-verifies each identity and fails closed, so a drifted image
stage is reported, never silently tolerated.

That container reproduces the Linux kernel and filesystem semantics, the 2-CPU
scheduling pressure the hosted runner applies, the `linux/amd64` build of every
package, and a cold module cache on first run. It does not reproduce hosted
runner hardware or disk speed (absolute wall times differ), the hosted image's
preinstalled tool surface beyond the pinned runtimes, or the macOS-only jobs
(`native-tests`, `golden-native`). The lane's Docker work runs against your host
daemon through the mounted socket rather than a nested daemon, so on a host of a
different architecture the pinned `linux/amd64` checker image is emulated
exactly as it is outside the container.

On an arm64 host such as Apple Silicon, an emulated `linux/amd64` race sweep is
impractically slow, and the BEAM's JIT traps under Rosetta: run `make ci-linux`
on a `linux/amd64` machine or VM for evidence. For an exploratory emulated run,
export `ERL_FLAGS="+JMsingle true"` (the script forwards it) or set
`MACHINERY_CI_LINUX_PLATFORM=linux/arm64`, and treat neither as hosted-CI
evidence.

The container runs on a pinned cpuset as well as a CPU quota, so runtimes that
size their schedulers from the visible core count see two cores, the way the
hosted runner does. Overrides: `MACHINERY_CI_LINUX_CPUS`,
`MACHINERY_CI_LINUX_CPUSET`, `MACHINERY_CI_LINUX_PLATFORM`, and
`MACHINERY_CI_LINUX_<STAGE>_IMAGE` per image stage (see
`scripts/ci-linux.dockerfile`).

## Bumping the linter

The golangci-lint version is the single source of truth in `.golangci-version`,
read by CI, `make lint-install`, and preflight. To bump it: edit that file, run
`make lint-install`, then `make preflight-fast`. If it is clean locally, CI's
lint job runs the identical binary.

## Versioning

Versions are numeric only (`vX.Y.Z`); local builds are not marked, so a bare
`go build` reports the same plain version as the release it corresponds to.

## Everything else

`make help` lists the contributor targets (build, test, golden, check,
verify-formal, and the gate targets above).
