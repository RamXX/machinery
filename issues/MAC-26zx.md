---
id: MAC-26zx
title: "integration-lane: cold module cache breaks native selection, and a nil error is formatted with %w"
status: closed
priority: 1
type: bug
labels: [ci, integration-lane, cold-cache]
created_at: 2026-09-08T13:03:43Z
created_by: ramirosalas
updated_at: 2026-09-08T13:28:34Z
content_hash: "sha256:8d01e86e62c16785ae8679d1cc8de405ced28713a7918b5d5ccd0d2d0210ab4e"
closed_at: 2026-09-08T13:28:33Z
close_reason: "Fixed at 399fbf0; required lane passes cold and warm, nil error never formatted with %w"
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
- 2026-09-08T13:28:34Z status: open -> closed

## Links


## Comments

### 2026-09-08T13:28:33Z ramirosalas
Fixed on fix/ci-mirror at 399fbf0 (lane) and 48514bf (tagged full-path contract).

Root cause sites:
- scripts/integration-lane/main.go:1413 (pre-fix): guard 'err != nil || errout != ""' with fmt.Errorf("native package selection failed: %w %s", err, errout); nil err renders as %!w(<nil>).
- Same nil-%w shape at main.go:1094 (toolReceipt) and assurance_catalog.go:636 (probe selection).
- No step warmed the module closure before selection, so a cold runner cache made 'go list' write 'go: downloading ...' to stderr.

Fix:
- warmGoModuleClosure (main.go) runs 'go mod download' once as a bounded (10m) custody-guarded job in the repo root before any selection, called from provision() right after the go tool receipt. Its stderr vocabulary is checked against goDownloadProgress (downloading/extracting/finding plus 'no module dependencies to download'); anything else fails closed. It emits its own receipt: id go-modules, identity go.mod+go.sum, sha256 over both manifests, no executable path.
- laneStreamFailure replaces every '%w %s' guard that can see a nil error; it describes a nil error and keeps a real one unwrappable.

Verification:
- go test -count=1 -run 'TestLaneStreamFailure|TestWarmGoModuleClosure' ./scripts/integration-lane -> ok 0.9s. The cold-cache test is offline: it serves the host module cache in proxy layout (file://$GOMODCACHE/cache/download) into an empty GOMODCACHE, asserts the selection stderr carries 'go: downloading' before warming and is empty after.
- go test -count=1 ./scripts/integration-lane -> ok 152.2s
- Full required lane, warm: go run ./scripts/integration-lane --lane required -> exit 0, '8 suites passed; 8 assurance suites passed across 4 native adapters', 4m02s, custody 60 jobs verified.
- Full required lane, cold (GOMODCACHE=$(mktemp -d)) -> exit 0, same 8+8, 5m01s. Both reports carry the go-modules receipt with the identical sha256.
- go test -count=1 ./cmd/machinery -> ok 266.1s
