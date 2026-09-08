---
id: MAC-ticd
title: "runtimeclosure: the Elixir tree fingerprint rejects the erlef/setup-beam OTP layout"
status: open
priority: 1
type: bug
labels: [ci, runtimeclosure, elixir]
created_at: 2026-09-08T13:04:01Z
created_by: ramirosalas
updated_at: 2026-09-08T13:28:53Z
content_hash: "sha256:456f777165de31a373ca3461fc56712f4cc489ef730985d949119b652bdf2c02"
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

### 2026-09-08T13:28:53Z ramirosalas
Fixed on fix/ci-mirror at 5f8d6a7.

Root cause site: internal/runtimeclosure/elixir.go:588 (pre-fix resolveElixirLink) -- filepath.Rel(rootPath, EvalSymlinks(link)) against the caller's unresolved root spelling, with a diagnostic that named only the link.

setup-beam layout found (erlef/setup-beam v1.23.0, src/setup-beam.js): darwin OTP comes from https://github.com/erlef/otp_builds/releases/download/OTP-29.0.6/OTP-29.0.6-macos-<arch>.tar.gz. In that tarball bin/epmd is the tree's only symlink and is relative: '../erts-17.0.6/bin/epmd'. installTool() caches the extracted dir, then does fs.cpSync(cachePath, $RUNNER_TEMP/.setup-beam/otp, {recursive:true}). Node's cpSync resolves a relative symlink against the SOURCE dir and writes it back absolute (verbatimSymlinks defaults false), so the installed link becomes $RUNNER_TOOL_CACHE/otp/OTP-29.0.6/<arch>/erts-17.0.6/bin/epmd -- outside the tree being fingerprinted. Reproduced byte-for-byte on this host: downloaded OTP-29.0.6-macos-arm64.tar.gz, ran node fs.cpSync over the extract, and the copy's bin/epmd is an absolute link into the source tree.

So the parent theory (symlinked root) was only half of it. Both halves are now fixed:
- fingerprintElixirTree resolves the root once (filepath.EvalSymlinks) and takes every containment decision against the resolved root, so an in-tree link reached through a symlinked root is in-tree.
- bindElixirLink replaces resolveElixirLink/readElixirLinked. An in-tree target keeps its root-relative identity (kind 'l'); an out-of-tree target is accepted and bound by the exact bytes it delivers under a distinct kind ('x') and a path-independent marker, so the closure digest stays install-path independent and still changes if the target's bytes change. Only an unresolvable, non-regular or over-bound target fails closed, and that diagnostic names the link and the resolved target.

Verification against the real artifact (fingerprintElixirTree over the unpacked tree):
- pre-fix, cpSync copy -> REJECTED 'symlink bin/epmd escapes the runtime tree' (the exact CI line).
- pre-fix, pristine tarball reached through a symlinked root -> same rejection.
- pre-fix, pristine tarball at its real path -> accepted.
- post-fix, all three accepted; the symlinked-root spelling yields the identical digest to the real path.

Tests: TestElixirTreeFingerprintAcceptsTheVendorLauncherLinkLayouts covers relative in-tree, absolute in-tree, in-tree through a symlinked root, and the setup-beam absolute link into a separate cache tree (including that tampering with the out-of-tree target changes the digest). TestElixirTreeFingerprintRejectsUnusableLinksWithBothPaths keeps dangling and directory targets closed and requires both paths in the diagnostic.

- go test -count=1 ./internal/runtimeclosure -run Elixir (Homebrew layout) -> ok 6.9s
- go test -count=1 ./internal/runtimeclosure -> ok 28.1s
