Package `assets/python` is the EXCLUSIVELY owned bounded executable-asset
directory of the closed `python-unittest/v1` adapter (MAC-imtz). Every asset
the adapter ships lives here; nothing else may add files to this tree.

## Exact bounded inventory (frozen before RED)

| Path | Role | Entry point | Dependencies |
|---|---|---|---|
| `machinery_check.py` | byte-pinned assertion helper / witness transport of `machinery-check/v1` | `check(case, assertion_id, condition)` | Python standard library only (`sys`) |
| `bootstrap.py` | byte-pinned embedded bootstrap/harness: closed `validate` / `discover` / `run` subcommands; static AST validation of the frozen suite; native TestLoader discovery with full `TestCase.id()` reconciliation; the embedded `TestResult` recording startTest/stopTest, addSuccess/addFailure/addError/addSkip, addExpectedFailure, addUnexpectedSuccess and addSubTest; the JSONL report stream with the `bootstrap-end` sentinel; stdlib identity anchors around import and execution; RuntimeWarning-as-error plus an unraisable hook for deterministic unawaited-coroutine residue | `python -I bootstrap.py validate\|discover <inventory>` / `run <inventory> <report>` | standard library only (`ast`, `gc`, `importlib`, `io`, `json`, `os`, `sys`, `types`, `unittest`, `warnings`) |
| `conformance/conformance_test.py` | frozen native conformance fixture suite executed by the required contributor lane fragment `testdata/integration-lanes/assurance-python.json` | `ConformanceAsync.test_async_witness` (IsolatedAsyncioTestCase), `ConformanceIdentity.test_class_identity`, `ConformanceMultiple.test_multiple`, `ConformanceWitness.test_witness_pass` | standard library `unittest` + `machinery_check` |

That is the complete executable/configuration closure: 3 files plus this
README (within the reviewed bound of at most 8 executable/configuration
assets plus README). There is no vendored runtime, no pytest, no doctest, no
custom load_tests/result/runner class, and no alternate filesystem copy of
any asset.

## Embedding owner and byte-source relation

`internal/tdd/adapters/python.go` is the ONLY embedding owner. It embeds
`machinery_check.py` and `bootstrap.py` with `go:embed`, pins their exact
sha256 as compile-time constants, materializes them into every prepared
suite at `<suite>/machinery_check.py` and `<suite>/_machinery_bootstrap.py`
(mode 0644), read-back verifies the materialized bytes against the pinned
digests before any native invocation, and re-verifies them after execution
(late mutation fails the run). No pre-existing `__pycache__` or `.pyc` is
accepted in the prepared suite tree and none may appear after the run. The
conformance fixture is NOT embedded: the lane fragment and the adapter's own
tests read its frozen bytes from this tree, hash-bind them, and capture them
through the real `tdd.Capture` bundle path before execution.

## Witness transport contract

`check(case, assertion_id, condition)` evaluates exactly one registered
assertion at its frozen call site (resolved with `sys._getframe(1)`), the
condition argument is strictly boolean (a non-boolean is a `TypeError`,
never an assertion outcome), and exactly one machine witness record

    {"schema":"machinery.tdd.witness/v1","kind":"witness",
     "assertion":<ID>,"test":<TestCase.id()>,"site":"<FILE>:<LINE>",
     "condition":<bool>,"thrown":null|"AssertionError"}

flows through the embedded harness channel (`sys.modules
["_machinery_harness"]`, registered only by the bootstrap before the suite
imports). On a false condition the witness is emitted first and then the
helper raises `AssertionError` itself so the native framework fails the
test; the helper never catches or converts a product exception. The only
supported call spellings are `machinery_check.check(case, "<id>", ...)` and
`from machinery_check import check` / `check(case, "<id>", ...)`; aliased
imports are rejected by static validation.

## Planned native identities

`conformance/witness-pass`, `conformance/class-identity`,
`conformance/async-witness`, `conformance/multi-a` and `conformance/multi-b`
are the registered assertion IDs of the frozen fixture; their exact call
sites are validated structurally (typed AST call targets at the registered
lines, inside the declared methods) by the embedded bootstrap's `validate`
subcommand under the pinned CPython 3.14.7 closure at Prepare time.
