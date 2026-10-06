---
id: MAC-8j82
title: "oracle: single-file mode deletes every sibling oracle (sequential, no concurrency needed); per-file publications also share one sentinel"
status: closed
priority: 0
type: bug
labels: [oracle, publication, concurrency, h2]
created_at: 2026-09-10T07:50:17Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:38Z
content_hash: "sha256:65051583d8310b28ddb9d0a8b20888d8cc3e1aa28300fb2856d45f43530db5d3"
closed_at: 2026-10-06T05:53:38Z
close_reason: "Fixed in 0.11.1 (9ae02d4f RED, 40cb4fb1 GREEN): oracle deletes only generated oracles whose source machine is gone; reproduced and verified with the real binary on a copied example (5 of 5 oracles preserved)."
---

## Description
Observed 2026-09-10 on H2 (design with 90 machines, machinery v0.7.2). Four agents ran `machinery oracle design/machines/<Name>.machine.json` on DIFFERENT files concurrently. The publication is directory-wide: the sentinel design/.machinery-design-publish.json lists every machines/*.oracle.md as an output even for a single-file invocation, and the interrupted run left ALL 90 oracle files absent from the working tree (git status: 90 ' D' entries), with recover reporting 'input inventory does not match the recorded publication inputs ... action: rerun-writer'. Every later check, verify-c4 and lint invocation refused with 'interrupted Machinery publication "oracle" prevents a consistent design snapshot'. Expected: a single-file oracle publication scopes its outputs to that file (or takes a per-file lock), so concurrent regeneration of different machines is safe, or the command refuses to start while another publication holds the directory instead of both proceeding and destroying the outputs. Recovery on H2 was a full `machinery oracle design/machines` rerun after all writers stopped. Also worth documenting in the skill: never run oracle regeneration concurrently within one design.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: reproduced WITHOUT concurrency. One sequential `machinery oracle design/machines/X.machine.json` on a copy of examples/fulfillment/design: 6 oracles before, 1 after, exit 0. Cause: cmd/machinery/oracle.go:403 staleOwnedOracles(artifactDir, artifacts) receives keep = only the selected files; canonicalOracleOwner (oracle.go:483-509) treats every machinery-generated oracle as owned-stale without checking that its source .machine.json is gone. Introduced fda20430 (2026-09-02). TestOracleSingleFileMode (oracle_test.go:207) checks a sibling is not generated, never that an existing one survives. Fix: in per-file mode, stale = generated oracles whose source machine is absent; regression test with a pre-existing sibling. Explains H2's 90 deleted oracles. Raised to P0: silent data loss in a released command; warrants 0.10.2. Concurrency lock is secondary.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: In per-file mode only stale-delete generated oracles whose source machine is absent; add regression test with a pre-existing sibling oracle; then consider a directory publication lock for concurrency. Evidence: Reproduced by reading code: cmd/machinery/oracle.go:403 calls staleOwnedOracles(artifactDir, artifacts) where artifacts holds only the named files in per-file mode; staleOwnedOracles (oracle.go:450-481) marks every machinery-generated *.oracle.md not in keep as stale via canonicalOracleOwner (oracle.go:483) with no check that the source .machine.json is gone, so siblings are scheduled for ExpectAbsent deletion. TestOracleSingleFileMode (oracle_test.go:207) only asserts the sibling is not generated, never that an existing one survives. No fix in git log or CHANGELOG through 0.11.0. Notes: Silent data loss in a shipped command (triage already says P0). Root cause also explains H2 90 deleted oracles.

## History
- 2026-10-06T05:53:38Z status: open -> closed

## Links


## Comments
