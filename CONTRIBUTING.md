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

## The pre-push gate

`make hooks` points git at `.githooks/pre-push`, which runs `scripts/preflight.sh`
before any push leaves your machine. Preflight mirrors `.github/workflows/ci.yml`
job for job, cheapest check first (gofmt, vet, golangci-lint, `go mod tidy`, the
docs gate, build, race tests, the golden corpus, the example gate suites, and the
go-crm impl suite). A formatting slip fails in under a second instead of after a
red CI run.

A clean preflight run means a green CI run: both run the identical golangci-lint
version, read from `.golangci-version`.

Run it on demand any time:

```sh
make preflight
```

(Not to be confused with `machinery preflight`, the end-user subcommand that
checks toolchain prerequisites. `make preflight` is the contributor CI mirror.)

There is no bypass variable: the hook runs the full gate on every push, and
hosted CI re-runs every gate on the pushed commit. Release publication is gated
separately on the exact commit carrying successful ci, formal, and security
runs (see `docs/release-policy.json`), so a red push can never publish.

## Bumping the linter

The golangci-lint version is the single source of truth in `.golangci-version`,
read by CI, `make lint-install`, and preflight. To bump it: edit that file, run
`make lint-install`, then `make preflight`. If it is clean locally, CI's lint job
runs the identical binary.

## Versioning

Versions are numeric only (`vX.Y.Z`); local builds are not marked, so a bare
`go build` reports the same plain version as the release it corresponds to.

## Everything else

`make help` lists the contributor targets (build, test, golden, check,
verify-formal, and the gate targets above).
