---
id: MAC-3iga
title: "Gd-idcite CLAUSES collector matches prose lines, not guard table rows, and reports them as malformed declarations"
status: open
priority: 0
type: bug
labels: [regression, gates, gd-idcite]
parent: MAC-ui8a
created_at: 2026-09-08T07:54:31Z
created_by: ramirosalas
updated_at: 2026-09-08T07:54:31Z
content_hash: "sha256:6e2ca1ca4dae843c9080a4eb0ac00833495a4726ba1c171617b01409767a6688"
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


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
