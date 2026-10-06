---
id: MAC-qxaa
title: "Gt-tests credits oracle ids in suites that skip themselves; report them as UNRUN"
status: open
priority: 1
type: bug
labels: [gt, assurance, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:52Z
content_hash: "sha256:35a55d798db066f2f4dd2a9b8ed63bf730126769c9b346d61acfdf20cd1def38"
blocks: [MAC-9azz]
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
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Implement the skip vocabulary per framework (ExUnit moduletag/tag skip, runner-config exclude, literal-false env gate; Go t.Skip at top, pytest skip, node test.skip or document as out of scope), the UNRUN finding class, and the opt-in content-hash out-of-band record with staleness. Evidence: internal/gates/oraclecov.go handles Go //go:build ignore (:209-213, :270) and Go test function validity (:957-960) but has no skip vocabulary: grep for skip/pending/@tag/@moduletag/exclude in oraclecov*.go finds only build-ignore handling. Elixir .exs files are parsed (:472, :603) with no moduletag skip handling. No UNRUN class exists; CHANGELOG 0.10.x-0.11.0 does not mention it. Notes: Correctness hole in a gate (credits obligations as bound that no wall runs). Blocks MAC-9azz (overlapping 'skipped suites credit nothing' AC); do qxaa first. No dependency on MAC-va30.

## History
- 2026-09-24T21:33:56Z dep_added: blocks MAC-9azz

## Links
- Blocks: [[MAC-9azz]]

## Comments
