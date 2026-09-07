Package `assets/typescript` is the EXCLUSIVELY owned bounded executable-asset
directory of the closed `node-test-typescript/v1` adapter (MAC-avfp). Every
asset the adapter ships lives here; nothing else may add files to this tree.

## Exact bounded inventory (frozen before RED)

| Path | Role | Entry point | Dependencies |
|---|---|---|---|
| `machinery-check.ts` | byte-pinned assertion helper / witness transport of `machinery-check/v1` | `check(t, assertionId, condition)` | `node:assert/strict` only (runtime); `node:test` types only |
| `node-ambient.d.ts` | ambient type closure of the supported node surface (`node:test` incl. `TestContext.diagnostic`/`skip`/`test`, `node:assert/strict`); the closed type surface only — the adapter still rejects every native skip/todo at reconciliation | ambient module declarations | none |
| `tsconfig.json` | frozen compiler configuration of record: `strict`, `module`/`moduleResolution` `nodenext`, `target` `es2023`, `noEmitOnError`, `sourceMap` + `inlineSources`, `incremental` off, `types` `[]`; the native tsc 7 compiler refuses an implicit tsconfig alongside command-line files (TS5112), so the adapter materializes this record under `assets/` and reproduces exactly this option set on its closed argv | materialized frozen record at `assets/tsconfig.json`; option set reproduced verbatim on the compile argv | none |
| `package.json` | frozen module identity of every prepared suite (`"type": "module"` = Node ESM) | none | none |
| `reporter.mjs` | embedded Machinery reporter: one bounded JSON line per native `node:test` reporter event plus the terminal `machinery:reporter:end` sentinel | default async-generator export consumed by `node --test --test-reporter` | none (no imports at all) |
| `conformance/conformance.ts` | frozen native conformance fixture suite executed by the required contributor lane fragment `testdata/integration-lanes/assurance-typescript.json` | `conformance witness executes native assertion`, `conformance parent identity` (fixed child `conformance nested identity`), `conformance multiple assertions` | `node:test` + `./machinery-check.js` |

That is the complete executable/configuration closure: 6 files plus this
README (within the reviewed bound of at most 8 executable/configuration
assets plus README). There is no vendored runtime, no npm indirection, no
loader, no Babel/tsx/Jest/Vitest substitution, and no alternate filesystem
copy of any asset.

## Embedding owner and byte-source relation

`internal/tdd/adapters/typescript.go` is the ONLY embedding owner. It embeds
this tree with `go:embed`, pins every asset's exact sha256 as compile-time
constants, materializes the pinned bytes into every prepared suite
(`package.json` at the compile root, the frozen `tsconfig.json` record and
the reporter under the private assets directory, the helper and its ambient
declarations next to every suite source directory), read-back verifies the
materialized bytes against the pinned digests before any native invocation,
and re-verifies them after execution (late mutation fails the run). The
conformance fixture is NOT given to the adapter as an embed input: the lane
fragment and the adapter's own tests read its frozen bytes from this tree,
hash-bind them, and capture them through the real `tdd.Capture` bundle path
before execution.

## Witness transport contract

`check` evaluates exactly one registered assertion at its frozen call site.
The condition argument is strictly boolean (a non-boolean is a `TypeError`,
never an assertion outcome). It emits exactly one machine witness object
through the native `node:test` diagnostic channel:

    {"schema":"machinery.tdd.witness/v1","assertion":<ID>,"phase":"evaluated",
     "site":<COMPILED entry file path>:<line>:<column>,"condition":<bool>,
     "thrown":null|"AssertionError"}

The `site` is the first stack frame outside the helper itself, i.e. the
compiled call site of `check` inside the suite's registered test; the adapter
maps it back to the frozen TypeScript source line through the compiler's own
verified `.js.map` (whose embedded `sourcesContent` must equal the frozen
bundle bytes) and rejects any mismatch against the registered call site. On a
false condition the helper lets the native `assert.ok` throw so the native
framework itself fails the test (`thrown: "AssertionError"` is recorded in
the witness before the error propagates); the helper never catches or
converts a product exception into an assertion outcome.

## Planned native identities

`conformance/witness`, `conformance/nested`, `conformance/multi-a` and
`conformance/multi-b` are the registered assertion IDs of the frozen fixture;
their exact TypeScript call lines (5, 10, 15, 16) are validated by
`internal/tdd/adapters/typescript.go` at Prepare time and re-bound at run
time through the witness site plus the verified source map.
