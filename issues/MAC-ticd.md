---
id: MAC-ticd
title: "runtimeclosure: the Elixir tree fingerprint rejects the erlef/setup-beam OTP layout"
status: open
priority: 1
type: bug
labels: [ci, runtimeclosure, elixir]
created_at: 2026-09-08T13:04:01Z
created_by: ramirosalas
updated_at: 2026-09-08T13:04:01Z
content_hash: "sha256:b145c2cde983aad608885ea2f733af9fb148c0bac691f69e7e167e4d655b07a3"
---

## Description
## Symptom

Every Elixir closure and adapter test fails on the hosted macOS runner, where
Erlang/OTP 29.0.6 is installed by erlef/setup-beam v1.23.0 at
`/Users/runner/work/_temp/.setup-beam/otp`. The local Homebrew layout passes,
so this is a first-hosted-run defect.

## Verbatim CI line

Run https://github.com/RamXX/machinery/actions/runs/34225004428 job
`native-tests`:

```
--- FAIL: TestElixirClosureOpensPinnedCatalog (1.34s)
    elixir_test.go:58: pinned Elixir closure did not open: UNSUPPORTED_VERSION: fingerprint the Erlang/OTP tree: symlink bin/epmd escapes the runtime tree
--- FAIL: TestElixirClosureExpectedDigestEnforced (0.12s)
--- FAIL: TestElixirClosureValidateProbesUnderCustody (0.16s)
--- FAIL: TestElixirClosureCloseIsPureRevalidation (0.63s)
FAIL	github.com/RamXX/machinery/internal/runtimeclosure	43.990s
```

## Cause

`resolveElixirLink` in `internal/runtimeclosure/elixir.go` required every
symlink in the runtime tree to resolve to a path lexically under the *caller's
spelling* of the root, and it printed only the link, never the resolved
target. Two separate layouts break it:

1. A root reached through a symlink. `filepath.EvalSymlinks` on the link
   returns a fully real path while the root spelling still contains the
   symlink, so `filepath.Rel` yields `../...` and every in-tree link reads as
   an escape.

2. The layout erlef/setup-beam actually produces. It downloads
   `OTP-29.0.6-macos-<arch>.tar.gz` from the erlef/otp_builds release, in
   which `bin/epmd` is a relative link to `../erts-17.0.6/bin/epmd`, caches it
   under `$RUNNER_TOOL_CACHE`, and then copies the cache to
   `$RUNNER_TEMP/.setup-beam/otp` with Node's `fs.cpSync`. `cpSync` rewrites
   every relative symlink to an *absolute* path into the source tree, so the
   installed `bin/epmd` points at
   `$RUNNER_TOOL_CACHE/otp/OTP-29.0.6/<arch>/erts-17.0.6/bin/epmd`, outside
   the tree being fingerprinted. Rejecting that outright refuses an unmodified
   vendor layout, and the in-tree copy of the same bytes is right there.

Reproduced locally on the real artifact: unpacking
`OTP-29.0.6-macos-arm64.tar.gz` and running `fs.cpSync` over it produces the
absolute out-of-tree `bin/epmd` link, and the pre-fix fingerprint rejects it
with exactly the CI line.

## Expected

The fingerprint resolves the root once and takes every containment decision
against the resolved root, so a link reached through a symlinked root is
in-tree. A link resolving outside the tree is accepted and bound by the exact
bytes it delivers, under an entry kind that cannot collide with an in-tree
binding. Only an unresolvable, non-regular or unbounded target fails closed,
and that diagnostic names both the link and the resolved target. The Homebrew
layout keeps passing.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
