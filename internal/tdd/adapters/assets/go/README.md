Package `assets/go` is the EXCLUSIVELY owned bounded executable-asset
directory of the closed `go-testing/v1` adapter (MAC-wi2u). Every asset the
adapter ships lives here; nothing else may add files to this tree.

## Exact bounded inventory (frozen before RED)

| Path | Role | Entry point | Dependencies |
|---|---|---|---|
| `machinerycheck/machinerycheck.go` | byte-pinned assertion helper / witness transport of `machinery-check/v1` | `Check(t *testing.T, id string, condition bool)` | Go standard library only (`fmt`, `os`, `path/filepath`, `runtime`, `testing`) |
| `conformance/go.mod` | frozen native conformance fixture module identity | module `machinery.test/conformance`, `go 1.27` | none (no external module dependencies) |
| `conformance/conformance_test.go` | frozen native conformance fixture suite executed by the required contributor lane fragment `testdata/integration-lanes/assurance-go.json` | `TestConformanceWitnessPass`, `TestConformanceSubtestIdentity` (fixed children `first`, `second`), `TestConformanceMultipleAssertions` | standard library + `machinery.test/conformance/machinerycheck` |

That is the complete executable/configuration closure: 3 files plus this
README (within the reviewed bound of at most 8 executable/configuration
assets plus README). There is no vendored runtime, reporter, formatter or
bootstrap beyond the helper above; the native reporter is `go test -json`
itself and the native runner is the pinned Go toolchain closure owned by
`internal/runtimeclosure/go.go`.

## Embedding owner and byte-source relation

`internal/tdd/adapters/go.go` is the ONLY embedding owner. It embeds
`machinerycheck/machinerycheck.go` with `go:embed`, pins its exact sha256 as
a compile-time constant, materializes it into every prepared suite at
`<module>/machinerycheck/machinerycheck.go` (mode 0644), read-back verifies
the materialized bytes against the embedded bytes and against the pinned
digest before any native invocation, and re-verifies them after execution
(late source mutation fails the run). The conformance fixture is NOT
embedded: the lane fragment and the adapter's own tests read its frozen
bytes from this tree, hash-bind them, and capture them through the real
`tdd.Capture` bundle path before execution.

## Witness transport contract

`machinerycheck.Check` evaluates exactly one registered assertion at its
frozen call site. It writes exactly one machine witness line to stdout

    machinery-check/v1 witness id=<ID> value=<true|false> test=<NAME> site=<FILE>:<LINE>

which the native runner captures inside the enclosing test's `go test -json`
output events, and on a false condition additionally reports the failure
natively through `testing.T.Errorf` naming the caller's frozen source site.
It never recovers, catches or converts panics or errors of surrounding test
code; direct `t.Fatal`/`t.Error`/`TestMain` usage remains outside the strict
assertion-evidence surface and is rejected by the adapter.

## Planned native identities

`conformance/witness-pass`, `conformance/subtest-first`,
`conformance/subtest-second`, `conformance/multi-a`, `conformance/multi-b`
are the registered assertion IDs of the frozen fixture; their exact call
sites are validated structurally (typed AST call targets) by
`internal/tdd/adapters/go.go` at Prepare time.
