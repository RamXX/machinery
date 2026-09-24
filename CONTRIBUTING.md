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
checker reproduction, the Soufflé parity lane for the shipped Datalog rules
(`datalog-parity`, reproducible with `make dagger-job JOB=datalog-parity`), and
the native macOS suite run in
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
Node 26.9.0 with TypeScript 7.0.2, CPython 3.14.7, and Elixir 1.20.4 on OTP
29.1.1. The lane re-verifies each identity and fails closed, so a drifted image
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

## Runtime pins

machinery stays on the latest release of every native runtime it pins
(Erlang/OTP and its erts, Elixir, Node, TypeScript, CPython, Temurin Java, Go)
for security reasons. Dependabot covers Go modules and actions; the runtimes
are covered by a drift report:

```sh
make runtime-pins   # pinned, host-installed and latest-upstream per runtime
```

It exits 1 when a pin is behind its latest upstream release. The upstream
column prints `offline` only when no index can be reached at all; when one
lookup fails while others answer, its cell names the failure (for example
`lookup failed: HTTP 403`) and a warning is printed. `-strict` exits 2 on any
unknown upstream version. The status column adds `host behind` (or `host
ahead`) when the runtime on your PATH is not the pinned version, which is what
makes the native suites fail locally; that exits 0 by default, and
`go run ./scripts/runtime-pins -host-strict` exits 3 on it. Elixir is looked
up through the GitHub releases API (set `GITHUB_TOKEN` to avoid the
unauthenticated rate limit) with builds.hex.pm as the fallback.
`.github/workflows/runtime-pins.yml` runs it with `-strict` every Monday and
opens or updates the issue "runtime pins behind upstream" with the table.
Bumping a pin is a normal weekly chore, not a release event.

Each pin is defined once, in `internal/runtimeclosure` (`RequiredOTPVersion`,
`RequiredErtsVersion`, `RequiredElixirVersion`, `RequiredNodeRelease`,
`RequiredTypeScriptVersion`, `RequiredPythonVersion`, `RequiredGoVersion`, and
`pinnedJavaVersion` with its build and banner date in `provision.go`). The
integration lane derives its catalog from those constants. The other files
that spell a version (the assurance-runtimes action, `scripts/ci-linux.dockerfile`,
`testdata/integration-lanes/*.json` and the contracts, `.java-runtime-pin`,
`go.mod`, README and docs) are held to them by `go test ./scripts/runtime-pins`
and `go test ./internal/runtimeclosure`, which name every site that disagrees.
For OTP, take the erts version from `otp_versions.table`; for Java, take the
archive checksums from the Adoptium API. Then run the suites:

```sh
go test ./scripts/runtime-pins ./internal/runtimeclosure/...   # every pin site agrees
go test -count=1 ./internal/runtimeclosure/... ./scripts/integration-lane/... ./internal/tdd/...
make test && make check && make golden && make verify-formal
```

The native suites run against the runtimes on your PATH and fail closed on any
other version, so upgrade the host first (for example `brew upgrade node`).

Java tracks the latest patch of the Temurin 21 LTS line. Temurin 25 LTS was
tried on 2026-09-22 (25.0.4.1+1): TLC results were identical, but every Alloy
run failed because JDK 25 prints JEP 472 restricted-method warnings for
kodkod's `System::load`, and the Alloy runner rejects any extra engine output
("alloy exec emitted unexpected success diagnostics"). The probe also needs
`stdin.encoding`, a property new in JDK 25, allowed. Moving to 25 is a
separate change that has to handle both.

## Release safety: consumer corpora

The bundled examples do not contain every construct real designs use, and a
wrong specification passes its own tests. v0.10.0 passed every internal suite
and still turned the largest consumer design from 0 blocking and 0 warnings
into 30 blocking and 429 warnings; it was withdrawn the same day. The internal
gates cannot catch that class of regression. A differential run over real
designs can.

The rule: before pushing or tagging any change that touches gates,
projections, declarations, or the skill's grammar, build the candidate and run
`consumer-diff` with the last release binary over every consumer design you
have access to. Use read-only clones and never commit their content (no
paths, names, findings, or excerpts) to this repository. Record the result in
the candidate's release notes: the new and resolved counts per gate and the
allow file used.

```sh
make build                                   # the candidate, in .bin/machinery
make consumer-diff OLD=<last-release-binary> NEW=.bin/machinery \
  DESIGN=<clone>/design [IMPL=<clone>] [ALLOW=<allow-file>]
```

`consumer-diff` runs `machinery check` with both binaries over private copies
of the design (the enclosing git work tree, so Ga-accept keeps its history),
with the same flags, and hashes the original before and after: a changed
original is an error. It prints new findings, resolved findings, and changed
gate verdicts. Messages are normalized (copy paths, temporary paths,
durations, and version strings), and `checked:` counts are never findings.
It exits 0 when every new blocking finding and every new warning matches an
allow entry, 1 when one does not, and 2 on a usage or execution error, such
as a crashing binary or a missing design. Notes never gate. `-json` prints
the same report as JSON.

The allow file has one entry per line: `<gate> <severity> /<regexp>/
<reason>`. The gate is the printed id (`Gx-trace`), its short form (`gx`), or
`*`. The severity is `error`, `drift`, or `warn`. The regexp is matched
against the normalized message as the report prints it. The reason is
mandatory and should name the release-note item that makes the finding
expected. Entries that explain nothing are listed as unused; delete them
rather than let the list grow.

Version-stamp skew is not a regression. When a design was regenerated by a
newer machinery than the baseline, the baseline prints a skew note and may
report drift. The report names the stamping versions for each run and tags
that run's DRIFT and skew note `[stamp-skew]`. Skew in the baseline run
lands under resolved and cannot fail the diff. Skew in the candidate run
means the generator output changed. Consumers must regenerate, so those
findings still gate and need an allow entry citing the release note.

## Versioning

Versions are numeric only (`vX.Y.Z`); local builds are not marked, so a bare
`go build` reports the same plain version as the release it corresponds to.

## Everything else

`make help` lists the contributor targets (build, test, golden, check,
verify-formal, and the gate targets above).
