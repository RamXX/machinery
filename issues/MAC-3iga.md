---
id: MAC-3iga
title: "Gd-idcite CLAUSES collector matches prose lines, not guard table rows, and reports them as malformed declarations"
status: closed
priority: 0
type: bug
labels: [regression, gates, gd-idcite]
parent: MAC-ui8a
created_at: 2026-09-08T07:54:31Z
created_by: ramirosalas
updated_at: 2026-09-08T09:27:49Z
content_hash: "sha256:d1489dc48573c8b14b54c59ebb07145fe8f21fd897d59061c35d8666c61e96a9"
closed_at: 2026-09-08T09:27:49Z
close_reason: "Landed on main (release 0.7.0 line) at 5c35463: 9db810a CLAUSES declarations read from contract rows only; H2 check 163 to 78 with the sibling fixes"
---

## Description
## Symptom

Gd-idcite's CLAUSES collector treats ANY line containing the substring `CLAUSES` as a guard-row
declaration, including ordinary prose in the matrix's "Invariant coverage (attested)" section.
Thirteen prose bullets in a design that was green on v0.6.11 are reported as malformed guard rows
on v0.7.0.

## Reproduction

Design: H2 (/Users/ramirosalas/workspace/H2) at commit c59bc81, worktree copy.

```
machinery check design            # v0.6.11 -> exit 0, "0 blocking finding(s)", design-green
machinery check design            # v0.7.0  -> exit 1, 163 blocking findings
```

Minimal shape. A matrix file contains, outside any markdown table, the prose bullet
(design/machines/CoverageGap.matrix.md:176, under `## Invariant coverage (attested)`):

```
- `coverage-gap-subject-backed` - `guardGapSubjectBacked` enforces all eight of its declared clauses, CLAUSES{exactly-one-subject, norm-declared-release-backing, ...}: the subject arity, the
  DECLARED approved-release backing in norm mode ...
```

and design/machines/ComplianceTask.matrix.md:141, a CLAUSES list wrapped across prose lines:

```
  provenance (the rule's `deadline.relative_to` anchor, ...) with `guardDerivationAttributed` gating it (all six clauses:
  CLAUSES{derivation-source-typed, deriving-principal-tenant-bound, evidencing-records-provenanced,
  task-and-records-same-tenant, assignee-same-tenant, event-obligation-anchor-recorded}), and the first is a
```

## Expected

Prose that mentions a guard's clause vocabulary is not a declaration. Only rows of the named-unit
contract table (kind column `guard`) declare clauses. v0.6.11 behaved this way; these files are
unchanged.

## Actual

```
  ERROR  CoverageGap.matrix.md:176: machine "CoverageGap" guard "- `coverage-gap-subject-backed` - `guardGapSubjectBacked` enforces all eight of its declared clauses, CLAUSES{exactly-one-subject, ...}: the subject arity, the": malformed CLAUSES declaration; require one named guard row with CLAUSES{...} and optional RETIRED{...}
  ERROR  ComplianceTask.matrix.md:141: machine "ComplianceTask" guard "CLAUSES{derivation-source-typed, deriving-principal-tenant-bound, evidencing-records-provenanced,": malformed CLAUSES declaration; ...
```

13 such errors. The reported "guard name" is a sentence fragment, which is itself the tell that
the line was never a declaration.

## Root cause (code reading)

`internal/gates/clauses.go:63-65`:

```go
if !strings.Contains(line, "CLAUSES") && !strings.Contains(line, "RETIRED{") {
    continue
}
```

Every remaining line is then run through `splitTableRow` and rejected at line 76 when
`len(cells) < 3 || cellAt(cells, 1) != "guard"`. A non-table line can never satisfy that, so the
gate reports "malformed declaration" for text the author never wrote as one.

The pre-filter needs to select markdown table rows in the named-unit contract table (the same rows
`collectClauseDecls` already validates), not any line containing the token.

## Secondary problem: diagnostic quality

Even for a genuinely malformed row, quoting a 300-character prose fragment as `guard %q` is not a
usable diagnostic. When cells < 3, say the line is not a table row.

## H2 impact

BLOCKER for the v0.7.0 upgrade. 13 of the 163 new blocking findings, in files the design cannot
usefully rewrite: the prose is the attested invariant-coverage narrative that other gates and the
judgment reports depend on. Working around this means either deleting explanatory prose or
inventing an escape, both of which degrade the design.

## Acceptance Criteria


## Design


## Notes
Line-number correction, verified at release commit d1f2264 in the main checkout (the assessment worktree has since moved): the prose pre-filter is internal/gates/clauses.go:67-68, the table-row rejection is :74, the error text is emitted at :77, and the companion 'owning oracle has no transition governed by this guard' at :120. The 63-65/76 numbers in the original body were read from the pre-move worktree; the code is identical.

## History
- 2026-09-08T09:27:49Z status: open -> closed

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-08T08:31:51Z ramirosalas
Fixed on branch fix/MAC-3iga-wcrd at 3e5ab40 (worktree /Users/ramirosalas/workspace/machinery-worktrees/fix-gates). Not merged, not pushed. Left open.

RULING (symptom 1, prose collected as declarations): REGRESSION, restored. `git show v0.6.11:internal/gates/clauses.go` collected a declaration only when the clauseDecl regex matched AND `strings.Split(line, "|")` had at least 3 parts AND cells[1] was non-empty; every other line was skipped SILENTLY. Table rows only, and no shape was ever reported as malformed. Commit 1b47643 (fix(MAC-olrx)) replaced that with a substring pre-filter on `CLAUSES` / `RETIRED{` and made every non-matching line an ERROR. 0.7.0 now requires a `CLAUSES{` or `RETIRED{` group on a line carrying at least two pipes before it judges anything, which is the v0.6.11 shape, and captures the name from that row.

Three further undocumented narrowings from the same commit fired on H2 and are restored:
- kind column forced to `guard`: v0.6.11 had no kind check. Restored (AnswerProof.matrix.md:14 and ScopeEntry.matrix.md:52 are `actor` rows that carry CLAUSES).
- `strings.Count(line, "CLAUSES") != 1` counted the English word: "reconciled to the CLAUSES declaration here. CLAUSES{...}" (TrustLedger.matrix.md:84) is one declaration plus prose. Now counts `CLAUSES{` groups.
- the detached-RETIRED test matched the English word: "a RETIRED site" in a negative-test sentence (Asset.matrix.md:46, SiteDevice.matrix.md:54). Now looks for `RETIRED{`.

KEPT (now documented in CHANGELOG 0.7.0 Changed): one row declares one vocabulary. Conversation.matrix.md:77 carries CLAUSES{approval-resolved-approved, approval-names-this-conversation} plus a note quoting `CLAUSES{}`; two groups on one row name no single vocabulary. That is the only CLAUSES finding H2 still receives (1 of 88).

RULING (symptom 2, "owning oracle has no transition governed by this guard"): introduced by 1b47643, story MAC-olrx (closed, accepted). AC #2/#3 accept machine-local ownership, so the tightening is deliberate and story-accepted, but NOTHING in the CHANGELOG 0.7.0 section documented it (grep of that section for clause/guard/owner returns only unrelated entries). It fired 31 times on H2, every time on a guard NO oracle in the design governs: insert-only machines whose invariant sits on the creation edge (AnswerFeedback `give` -> applied carries an empty guard cell; DomainPack's oracle has no guard values at all). Verified all 31 names are absent from every design/machines/*.oracle.md in H2. v0.6.11's checkClauseCoverage pooled rows across all oracles, so such a declaration carried zero obligations and no finding.

Fix: keep the ownership property, drop the extra requirement. The error now fires only when a SIBLING machine's oracle governs the guard (the exact defect MAC-olrx closed: a declaration resolving through another machine), and names that sibling. A guard no oracle governs is silent, as in v0.6.11. TestObligationClausesOwnerCannotResolveThroughSibling/wrong-owner-guard passes unchanged, and TestClauseGuardGovernedOnlyBySiblingStillFails pins it directly.

Also: the quoted name in clause diagnostics is clipped (clipText), and a line that is not a contract row gets its own message instead of "malformed CLAUSES declaration".

Regression tests (internal/gates/clauses_test.go, all confirmed RED before the fix):
- TestClauseProseMentionIsNotADeclaration (H2's indented continuation-line shape)
- TestClauseDeclarationRowAcceptsProseAroundIt (English RETIRED, bare word CLAUSES, non-guard unit)
- TestClauseAmbiguousRowStillFails (two groups, detached RETIRED{): the retained tightening
- TestClauseGuardGovernedByNoOracleIsAccepted, TestClauseGuardGovernedOnlyBySiblingStillFails

Verification: go test -race -count=1 ./internal/gates -timeout 15m PASS 89s; go test -count=1 -run 'Golden|Check|Gate|Clause|Reads' ./cmd/machinery PASS 55s (golden corpus unchanged, no golden-update needed); gofmt clean; go vet clean; golangci-lint run --config .golangci.yml 0 issues.

H2 consumer check (read-only, commit c59bc81, machinery check design --warnings-as-errors): 163 blocking before, 88 after. CLAUSES family 44 -> 1 (13 malformed-prose + 31 oracle-ownership collapse to the one genuinely ambiguous row).
