---
id: MAC-8j82
title: "oracle: single-file mode deletes every sibling oracle (sequential, no concurrency needed); per-file publications also share one sentinel"
status: open
priority: 0
type: bug
labels: [oracle, publication, concurrency, h2]
created_at: 2026-09-10T07:50:17Z
created_by: ramirosalas
updated_at: 2026-09-24T21:31:55Z
content_hash: "sha256:356c550d7f98b3bbb88da0816ff13020c3bc704fd8bb77693f202891bcc375ad"
---

## Description
Observed 2026-09-10 on H2 (design with 90 machines, machinery v0.7.2). Four agents ran `machinery oracle design/machines/<Name>.machine.json` on DIFFERENT files concurrently. The publication is directory-wide: the sentinel design/.machinery-design-publish.json lists every machines/*.oracle.md as an output even for a single-file invocation, and the interrupted run left ALL 90 oracle files absent from the working tree (git status: 90 ' D' entries), with recover reporting 'input inventory does not match the recorded publication inputs ... action: rerun-writer'. Every later check, verify-c4 and lint invocation refused with 'interrupted Machinery publication "oracle" prevents a consistent design snapshot'. Expected: a single-file oracle publication scopes its outputs to that file (or takes a per-file lock), so concurrent regeneration of different machines is safe, or the command refuses to start while another publication holds the directory instead of both proceeding and destroying the outputs. Recovery on H2 was a full `machinery oracle design/machines` rerun after all writers stopped. Also worth documenting in the skill: never run oracle regeneration concurrently within one design.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: reproduced WITHOUT concurrency. One sequential `machinery oracle design/machines/X.machine.json` on a copy of examples/fulfillment/design: 6 oracles before, 1 after, exit 0. Cause: cmd/machinery/oracle.go:403 staleOwnedOracles(artifactDir, artifacts) receives keep = only the selected files; canonicalOracleOwner (oracle.go:483-509) treats every machinery-generated oracle as owned-stale without checking that its source .machine.json is gone. Introduced fda20430 (2026-09-02). TestOracleSingleFileMode (oracle_test.go:207) checks a sibling is not generated, never that an existing one survives. Fix: in per-file mode, stale = generated oracles whose source machine is absent; regression test with a pre-existing sibling. Explains H2's 90 deleted oracles. Raised to P0: silent data loss in a released command; warrants 0.10.2. Concurrency lock is secondary.

## History


## Links


## Comments
