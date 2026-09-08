---
id: MAC-26zx
title: "integration-lane: cold module cache breaks native selection, and a nil error is formatted with %w"
status: open
priority: 1
type: bug
labels: [ci, integration-lane, cold-cache]
created_at: 2026-09-08T13:03:43Z
created_by: ramirosalas
updated_at: 2026-09-08T13:03:43Z
content_hash: "sha256:db5e834b069102da2bf20e7961bc83db92c06ebaf90fb0b34d126a276d1b1832"
---

## Description
## Symptom

The `integration-required` job fails on a fresh hosted runner, in both the ci
and formal workflows, before any suite executes. The lane aborts on the first
suite's native package selection and prints a formatting placeholder where the
cause should be.

## Verbatim CI line

Run https://github.com/RamXX/machinery/actions/runs/34225004428 job
`integration-required` (identical in run 34225004430):

```
checker-oci-lifecycle: native package selection failed: %!w(<nil>) go: downloading github.com/spf13/cobra v1.10.2
go: downloading github.com/pelletier/go-toml/v2 v2.4.3
go: downloading github.com/spf13/pflag v1.0.10

exit status 1
```

## Cause

Two independent defects, both first observable on a cold module cache.

1. `scripts/integration-lane/main.go` runs `go list -json -tags
   machinery_integration <pkg>` to select each go-json suite and treats any
   stderr as failure. On a fresh runner the module cache is cold, so that
   first `go list` resolves imports and writes `go: downloading ...` progress
   to stderr. The local preflight never sees this because the host cache is
   warm. Nothing warms the closure before selection.

2. The guard is `if err != nil || errout != ""` but the diagnostic is
   `fmt.Errorf("native package selection failed: %w %s", err, errout)`. When
   only the stderr condition trips, `err` is nil and `%w` renders as
   `%!w(<nil>)`, destroying the operator's evidence. The same shape exists at
   `toolReceipt` (main.go), the docker inspect guard, and the assurance probe
   selection in `scripts/integration-lane/assurance_catalog.go`.

## Expected

The lane warms the main module's dependency closure once, in a bounded,
custody-guarded job with its own receipt, before any selection runs, and it
checks that warm step's stderr vocabulary rather than ignoring stderr. No
diagnostic ever formats a nil error through `%w`. A cold-cache selection
reproduction and the nil-error diagnostic are covered by tests in
`scripts/integration-lane`.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
