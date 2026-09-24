# Required assurance lane catalog contract

This is the explicit RED test contract for MAC-bz1y, the four-language native
assurance conformance catalog of the required integration lane. It extends the
frozen v1 lane contract (CONTRACT.md) compatibly; it does not rewrite any
frozen v1 file, fragment or test.

## Closed catalog

`testdata/integration-lanes/assurance.schema.json` (hash-pinned in the runner)
defines the closed assurance fragment schema `machinery.assurance.lane/v1`.
`testdata/integration-lanes/assurance-runtime-pins.json` declares the exact
first-release native catalog: go-testing/v1 Go 1.27.1; node-test-typescript/v1
Node 26.8.1 with TypeScript 7.0.2; python-unittest/v1 CPython 3.14.7;
elixir-exunit/v1 Elixir 1.20.4 with Mix 1.20.4, OTP major 29 and ERTS 17.0.6
(the OTP patch identity is bound through the runtime's own reported ERTS
version). Git 2.55.0 is declared as a gate runtime, never a fifth test
adapter. Both native platforms, darwin/arm64 and linux/amd64, are pinned.

The catalog activates for a lane root when any assurance-owned file is present
under testdata/integration-lanes (assurance.schema.json,
assurance-runtime-pins.json, assurance-*.json fragments or the
assurance-probes fixture tree). For the machinery repository itself (the lane
root whose go.mod declares module github.com/RamXX/machinery) the catalog is
mandatory: removing every assurance file fails the lane with an explicit
diagnostic instead of silently reverting to v1-only semantics. Foreign roots
(the frozen v1 fixtures and the full-path meta-test module) keep exact v1
interpretation.

## Fail-closed enforcement

When active, before any assurance suite executes, the catalog validates and
then provisions:

- The pins file equals the exact closed catalog values; any tampered version,
  missing adapter, missing git gate runtime or wrong platform set fails.
- The current GOOS/GOARCH pair is one of the pinned native platforms;
  anything else fails UNSUPPORTED_PLATFORM rather than degrading.
- Every one of the four adapter runtimes is really present and its native
  version identity matches the pin exactly: `go version` (go1.27.1), `node
  --version` (v26.8.1), `tsc --version` (Version 7.0.2), `python3 --version`
  (Python 3.14.7), `elixir --version` (Elixir 1.20.4, Erlang/OTP 29,
  erts-17.0.6) and `mix --version` (Mix 1.20.4). Each receipt binds the
  resolved executable bytes by sha256. A missing runtime, a mismatched
  identity, or an extra undeclared language fails closed; no runtime is
  fetched or installed during replay, and an arbitrary PATH shim is not a
  valid closure (full immutable closure digests are the downstream adapter
  stories' RuntimeHandle obligation; this catalog binds version identity and
  executable bytes).
- The fragment union is complete: each of the four adapter IDs owns at least
  one suite across all assurance-*.json fragments. An omitted language, an
  unknown adapter, a duplicate suite ID, a duplicate native test identity, an
  empty case inventory, a probe case outside the closed per-adapter probe
  inventory, or an unregistered file inside the assurance-probes fixture tree
  fails before any suite runs.
- Git 2.55.0 is verified the same way only once a suite declares the git gate
  runtime; no current suite does, matching the scoped-Git producer story.

## Catalog probe fixtures

This story ships one fragment, assurance-probes.json, owned by MAC-bz1y. Each
suite names actual runnable native cases in the frozen fixture tree
testdata/integration-lanes/assurance-probes (the go test function, the
TypeScript node:test case, the exact Python TestCase.id(), and the ExUnit
test name). The runner copies the exact frozen bytes into a private scratch
module, compiles where the language requires it, and executes real native
invocations: `go test -json -count=1`, a pinned tsc 7.0.2 compilation step
followed by `node --test --test-reporter=tap` on only that compiled output,
`python3 -I -m unittest discover -v`, and `mix test --trace --seed 0
--max-cases 1` with a private MIX_HOME/MIX_BUILD_PATH/HOME and no network.
These are runtime conformance probes of the catalog, not the downstream
adapters' assertion transports; each downstream adapter story (MAC-wi2u Go,
MAC-avfp TypeScript, MAC-imtz Python, MAC-8yai Elixir) owns its NEW named
fragment and extends this closed schema through an independently reviewed
compatible revision, never by editing frozen files.

Every suite accounts exact selected/started/passed/failed/skipped counts from
the native event stream itself; skipped, failed, extra or missing cases
cannot become success, and retained events are hashed into the report. The
probe executions are guarded custody jobs of the same lane root, with the
same bounded outputs, and the report gains one closed assurance section
(schema, status, platform, per-adapter receipts, runtime receipts, suite
receipts) without altering any v1 report field.

## Evidence limits

Structural validation proves the closed catalog, not runtime correctness.
Only real native processes executing the frozen probe bytes under the four
pinned language runtimes establish conformance evidence, on darwin/arm64 and
linux/amd64 hosted runners and the final local preflight. Full preflight
remains deferred until final epic completion.
