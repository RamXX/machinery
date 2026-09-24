---
id: MAC-qxaa
title: "Gt-tests credits oracle ids in suites that skip themselves; report them as UNRUN"
status: open
priority: 1
type: bug
labels: [gt, assurance, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:50Z
content_hash: "sha256:94f0f105badaf8fd411dacb4bc3ed413435e999f815ba8154b15e37045a6dbff"
---

## Description
Gt-tests credits oracle ids in suites that skip themselves; report them as UNRUN

Problem (H2, 2026-09-20): six suites run the real conversion reader as container tool steps and carry `@moduletag :reader_image`; they skip themselves wherever the pinned image is not served, which is every CI container. `Gt-tests` credits them by static discovery of their oracle-id tokens and executes nothing, so the design read those obligations as bound while no wall ever ran the tests. A fresh-context acceptance review found it, not a gate. H2 built its own instrument (`tools/ci/reader_record_gate.sh`, a content-hash record of the last green out-of-band run, refused on drift), a workaround for a gap the gate should own.

Evidence: internal/gates/oraclecov.go handles Go build-ignore files (:209-213) and Go test-function validity (:957), but has no ExUnit tag handling (`@moduletag skip`, `@tag :skip`, `exclude` in `test_helper.exs`) or equivalent for pytest/node skip markers. Related: MAC-9azz (open) requires "skipped suites credit nothing" for table oracles only.

Proposed fix:
- Gt reads the skip vocabulary of each framework it credits: module and test tags `skip` and `pending`, an `exclude` list in the runner config (ExUnit `ExUnit.configure(exclude: ...)`, `ExUnit.start(exclude: ...)`), and an environment-gated `@moduletag` whose condition is a literal false in CI. A credited id whose only binding test is skipped is reported UNRUN, a finding class distinct from unbound.
- An opt-in "out-of-band record" row type binding such a suite to a recorded run by content hash, so H2's workaround becomes a machinery contract. The record stales when the suite or its named sources change.

Acceptance criteria:
1. A suite with `@moduletag skip: "..."` credits nothing; Gt reports each of its ids as UNRUN with the file path.
2. A suite excluded by tag in the runner config credits nothing and reports UNRUN.
3. The same skipped suite with a committed out-of-band record binding its content hash credits its ids.
4. The record stales (blocking finding) when the suite bytes or any named source changes.
5. Equivalent skip markers for at least the other languages Gt credits (Go `t.Skip` at function top, pytest `skip` markers, node `test.skip`) are covered, or explicitly listed as out of scope in the docs.
6. Existing Gt golden output for suites without skips is unchanged.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
