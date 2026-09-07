Package `assets/elixir` is the EXCLUSIVELY owned bounded executable-asset
directory of the closed `elixir-exunit/v1` adapter (MAC-8yai). Every asset
the adapter ships lives here; nothing else may add files to this tree.

## Exact bounded inventory (frozen before RED)

| Path | Role | Entry point | Dependencies |
|---|---|---|---|
| `mix.exs` | frozen harness Mix project definition of every prepared suite (`machinery_assurance_harness`, no dependencies, no aliases, no project-supplied reporter/formatter hooks) | none | Elixir/Mix 1.20.4 standard libraries only |
| `machinery/json.exs` | bounded JSON codec of the closed reporter vocabulary (byte-wise escaping of quote, backslash and C0 controls; hard cut bounds on every string) | `escape/2`, `kv/1`, `line/2` | none |
| `machinery/reporter.exs` | embedded Machinery reporter: the named `Machinery.Assurance.Reporter` GenServer is simultaneously the sole ExUnit formatter (lifecycle casts from `ExUnit.Runner`) and the synchronous witness collector of the `machinery-check/v1` transport; appends one bounded JSON line per event to the private events file named by `MACHINERY_ASSURANCE_EVENTS`, terminated by the closed `machinery:reporter:end` sentinel after `suite_finished`; reports the effective ExUnit options (`seed`, `max_cases`, `trace`, `timeout`, `max_failures`, `include`, `exclude`, `formatters`, `dry_run`, `repeat_until_failure`) from the runtime's own `{:suite_started, opts}` cast; classifies native outcomes (`passed`/`failed`/`skipped`/`excluded`/`invalid` + failure class with `ExUnit.AssertionError` distinguished from raises/throws/exits/timeouts) | `GenServer` callbacks | `ExUnit`, `Machinery.Assurance.JSON` |
| `machinery/check.exs` | byte-pinned `machinery-check/v1` assertion helper / witness transport: `Machinery.Check.check/3` is a macro so the registered call site is the exact frozen test source line; the strictly-boolean condition is evaluated exactly once, every evaluation emits exactly one synchronous witness bound to the framework-owned context identity (module, describe, test, framework-recorded test pid), and a false condition raises a native `ExUnit.AssertionError` naming the frozen site after the witness is recorded | `Machinery.Check.check(ctx, assertion_id, condition)` | `ExUnit`, `Machinery.Assurance.Reporter` |
| `test/test_helper.exs` | frozen embedded bootstrap: fixed load order (codec, reporter, transport) then `ExUnit.start` with the closed formatter replacement; effective options are merged from the frozen `mix test` argv after this file evaluates and are reported by the embedded reporter itself | required by `mix test` before any test file loads | `machinery/*.exs` |
| `conformance/conformance_test.exs` | frozen native conformance fixture suite executed by the required contributor lane fragment `testdata/integration-lanes/assurance-elixir.json` | `conformance witness executes native assertion`, describe parent `conformance parent identity` with `conformance nested identity`, `conformance multiple assertions` | `ExUnit.Case` + `Machinery.Check` |

That is the complete executable/configuration closure: 6 files plus this
README (within the reviewed bound of at most 8 executable/configuration
assets plus README). There is no vendored BEAM runtime, no Hex dependency,
no project mix alias, no user formatter and no alternate filesystem copy of
any asset. The harness project has no `mix.lock` because it declares zero
dependencies and Mix never writes one for it; offline-ness is structural.

## Embedding owner and byte-source relation

`internal/tdd/adapters/elixir.go` is the ONLY embedding owner. It embeds
this tree with `go:embed`, pins every asset's exact sha256 as compile-time
constants, materializes the pinned bytes into every prepared suite (the
harness `mix.exs`, `machinery/` transport and `test/test_helper.exs`),
read-back verifies the materialized bytes against the pinned digests before
any native invocation, and re-verifies them after execution (late mutation
fails the run). The conformance fixture is NOT given to the adapter as an
embed input: the lane fragment and the adapter's own tests read its frozen
bytes from this tree, hash-bind them, and capture them through the real
`tdd.Capture` bundle path before execution. The declared frozen test files
of a suite are copied into the harness `test/` directory; no undeclared
file can execute because the frozen argv selects exactly the declared
files.

## Witness transport contract

`Machinery.Check.check` evaluates exactly one registered assertion at its
frozen call site. The condition argument is strictly boolean (a non-boolean
raises `ArgumentError`, never an assertion outcome). It emits exactly one
machine witness line through the reporter:

    {"module":<atom>,"describe":<string>,"test":<atom>,"id":<ID>,
     "value":<bool>,"file":<relative frozen path>,"line":<int>,
     "verdict":"ok"|"pid-mismatch"|"malformed"}

The identity comes from the framework-owned ExUnit context; a context whose
framework-recorded test pid is not the calling process is reported as
`pid-mismatch` and rejected by reconciliation. On a false condition the
helper raises the native `ExUnit.AssertionError` AFTER the witness is
recorded so the native framework itself fails the test; the helper never
catches or converts a product exception into an assertion outcome. Static
Prepare-time validation requires the registered line's trimmed content to
BEGIN with the typed `Machinery.Check.check(` call and carry the exact
assertion-id literal on the same line, so a commented-out `#` call can
never satisfy registration; runtime evidence is always execution-derived,
never comment-derived. Assertions in setup/setup_all/on_exit, throws,
exits, ordinary raises and ExUnit test timeouts remain error classes, never
assertion RED.

## Effective native options and argv

The frozen argv is `mix test --seed <adapter seed> --max-cases <adapter
max_cases> --warnings-as-errors <declared files...>` run with a fresh
private `MIX_BUILD_PATH`, private `MIX_HOME`/`HOME`/`TMPDIR`, an offline
closed environment and the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS 17.0.6)
runtime closure on `PATH`. Reconciliation rejects any effective-option
drift the reporter reports (filters, formatter replacement, truncation,
re-run or dry-run modes).

## Planned native identities

`conformance/witness`, `conformance/nested`, `conformance/multi-a` and
`conformance/multi-b` are the registered assertion IDs of the frozen
fixture; their exact Elixir call lines (8, 13, 18, 19) are validated by
`internal/tdd/adapters/elixir.go` at Prepare time and re-bound at run time
through the witness site. Native test identities are the exact
module/name/file/line the framework reports (the describe'd name carries
the describe prefix, e.g. `conformance parent identity conformance nested
identity`), never prettified display labels.
