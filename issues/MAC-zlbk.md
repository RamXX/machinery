---
id: MAC-zlbk
title: "One-step re-judgment: machinery attest --rejudge refreshes cover hashes and appends a dated note"
status: open
priority: 2
type: feature
labels: [attest, gv, ux, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:0268aa99796e03eb8cc8846fcf884a6800b683f53b9aeb5bbaf3672433a30be5"
related: [MAC-6mzy, MAC-mkh1, MAC-nf0m]
---

## Description
One-step re-judgment: machinery attest --rejudge refreshes a row's cover hashes and appends a dated note

Problem (H2, 2026-09-20): every prose edit to a design source stales four to ten attestation rows, and the remedy is manual: run `machinery attest <paths>`, paste hashes into the right rows, prepend a `RE-JUDGED <date> ...` line to each note. H2's record has rows with a dozen RE-JUDGED lines, three of them byte-identical duplicates from one day, and the judgment text almost never changes because a one-line prose edit does not move a placement or a guard. Eleven regenerations in one day, three to five minutes of conductor attention each, for zero changed judgments.

Evidence: `machinery attest` flags are `--claims --design --impl --claim --kind --attestor --date --note` (cmd/machinery/attest.go:92-99); no rejudge mode, no section-scoped covers, no duplicate-note lint.

Proposed fix:
- `machinery attest --rejudge <claim> --note "<text>"`: refresh every cover hash of that row from the tree and append the dated line in one step, editing the record in place.
- Covers scoped to a section or stable id instead of a whole file where meaningful (`g4.zero-context` over `BUILD.md` moves on every dated paragraph; the reviewer re-reads a paragraph, not the file).
- A lint that refuses a duplicate RE-JUDGED line in a row's note.

Acceptance criteria:
1. `machinery attest --rejudge <claim> --note "..."` updates every cover hash of that row and prepends one dated line; all other rows are byte-identical.
2. Running it twice on the same day with the same note is refused or is a no-op; a row never carries the same dated line twice, and Gv (or lint) reports a pre-existing duplicate.
3. A section-scoped cover stays fresh when a different section of the same file changes and stales when its section changes.
4. The eleven H2 refreshes of 2026-09-20 are each expressible as one command (documented example).

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add attest --rejudge <claim> --note (in-place refresh of cover hashes plus one dated line, idempotent same-day), a duplicate RE-JUDGED lint, and section-scoped covers. Evidence: Verified at HEAD 8c620d8f (v0.11.0). cmd/machinery/attest.go:92-99 still lists only --claims --design --impl --claim --kind --attestor --date --note; grep for rejudge/RE-JUDGED finds nothing in cmd, internal or skills (only test fixtures in example_attestation_migration_test.go). CHANGELOG 0.11.0 does not add it. Commit f2f9410a only re-pinned go-crm dates by hand. Notes: Section-scoped covers overlap MAC-nf0m (derived per-claim scope) and MAC-6mzy; consider splitting the section-scope AC out so --rejudge can ship alone. Related MAC-mkh1 (--merge-into) is a natural companion, not a prerequisite.

## History


## Links
- Related: [[MAC-6mzy]], [[MAC-mkh1]], [[MAC-nf0m]]

## Comments
