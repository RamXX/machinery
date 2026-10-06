# Gt test discovery and skipped bindings

`machinery check <design> --impl <implementation> --gate gt` discovers
stable oracle ids in test bodies. It does not execute tests. A reference in
an active test still establishes discovery only.

Gt excludes recognized skipped tests from both literal-id credit and whole
oracle parser credit. When an id has only skipped bindings, the blocking
finding is `UNRUN <stable-id>` and lists the implementation-relative test
paths. An id with no binding keeps the existing missing-id finding. An active
binding takes precedence over a skipped binding for the same id.
`OracleBindings` uses the same rules, so skipped tests do not enter exported
bindings.

## Recognized skip vocabulary

- ExUnit: `@moduletag`, `@describetag`, and `@tag` with atom or keyword `skip`
  or `pending`. Module and describe tags stay within their block; test tags
  apply to the next test. Literal `false` and `nil` do not skip. Literal
  `not false` and `!false` skip. Other skip expressions, including environment
  checks, conservatively produce UNRUN because static discovery cannot prove
  that the runner enables them. Module tags also apply to tests declared
  earlier in the same module.
- ExUnit runner configuration: literal `exclude: [...]` lists in
  `ExUnit.start(...)` or `ExUnit.configure(...)` in `test_helper.exs` anywhere
  under the implementation root. Atom entries exclude matching tags;
  keyword entries exclude matching literal tag values.
- Go: a direct `Skip`, `Skipf`, or `SkipNow` call on the test parameter as
  the first statement of a valid test function.
- pytest: single-line `@pytest.mark.skip` decorators and `pytestmark`
  assignments, including single-line marker lists. A literal
  `pytest.mark.skipif(True, ...)` is also recognized.
- Node test/Jest style: `test.skip`, `it.skip`, `test.todo`, and `it.todo`,
  including `.skip.each` data-driven calls in the JS/TS extensions Gt scans.

The following skip structures remain outside this static vocabulary:
conditional or helper-mediated Go skips, Python class decorators and
multiline markers, nonliteral `skipif` conditions, node skipped containers
and runner options, dynamically assembled ExUnit exclusion lists or tags,
and Ruby/RSpec and Rust skip/ignore markers. Such forms do not receive the
UNRUN classification from this feature. Use supported explicit test markers
when the distinction is required. Gt's existing parser limitations still
apply; skip detection does not add runtime execution or a language compiler.

## Out-of-band run records

A suite intentionally skipped by the normal runner can opt in to a recorded
passed run. Commit `<design>/assurance/test-runs.json` as a JSON array:

```json
[
  {
    "type": "out-of-band",
    "suite": "test/reader_test.exs",
    "sha256": "<64 lowercase hexadecimal characters>",
    "sources": {
      "lib/reader.ex": "<64 lowercase hexadecimal characters>",
      "test/test_helper.exs": "<64 lowercase hexadecimal characters>"
    },
    "result": "passed"
  }
]
```

Each row binds one suite file to a passed run of its exact bytes. `sha256`
is the SHA-256 of the complete suite bytes, without normalization. `sources`
is a required object containing the SHA-256 of every named dependency of
that run; use an empty object when there are none. Include runner
configuration and other inputs that affect the suite. All paths are clean,
slash-separated paths relative to `--impl` and must belong to the governed
implementation inventory. Symlinks, paths outside that root, and sources
excluded from the inventory cannot establish a record.

After running the suite out of band successfully, compute hashes with
`sha256sum <suite> <sources...>` (or `shasum -a 256`), fill the row, and commit
it with the corresponding files. The row is an attestation of a passed run;
Gt verifies its byte bindings, not the execution or truth of that assertion.
A matching record restores the suite's existing literal and parser discovery
credit, including exported `OracleBindings`. It cannot manufacture bindings
absent from that suite's extracted test bodies.

Changed suite bytes, changed or missing named sources, malformed records,
unknown fields, duplicate suite rows, and results other than `passed` block
Gt. Stale or invalid rows restore no credit, so skipped bindings also remain
UNRUN. Re-run the suite before replacing stale hashes. Designs without this
file and suites without recognized skips retain their existing Gt output.
