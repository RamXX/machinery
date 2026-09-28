---
id: MAC-vfi4
title: "Gt: first-class owed-by-milestone declaration for oracle rows and clauses"
status: open
priority: 1
type: feature
labels: [hard-tdd, gt, owner-request]
created_at: 2026-09-28T15:49:09Z
created_by: ramirosalas
updated_at: 2026-09-28T15:49:09Z
content_hash: "sha256:8adb1e78477f2444a0252170139debb23cd85caef85b6646047c65036507acae"
---

## Description
## USER INTENT
A design can land a new machine whose oracle rows and falsifying clauses are owed by an open build milestone, without weakening Gt for anything else, and without project-local exemption lists. Owner request, 2026-09-28, from the NIL composition v2 design (NIL nd NI-bbu1).

## Problem (reproduced on v0.10.1)
- Gt-tests fails whenever any committed oracle row, or any declared CLAUSES id, has no active test reference.
- A brownfield design that adds new machines for work the BUILD.md plan assigns to open milestones cannot land on the implementation repo's main without turning the release gate red. The tests can only be written by those milestones' RED stories.
- None of these help:
  - BUILD.md oracle-binding `unbound` rows are checked by Gy but do not exempt Gt.
  - `machinery baseline --gate gt` refuses.
  - Adjudication covers characterization only.
  - Gw waivers apply to slices only.
  - docs/test-assurance-contract.md describes the needed semantics, but no gate reads design/assurance/.
  - The wave sentinel exists only in the Claude plugin hook.
- The NIL workaround (owner-authorized option B): a committed, shrink-only list of owed ids in the project's gate script.

## Target behavior (to design properly here)
- A first-class, gate-read declaration that binds specific oracle stable ids and clause ids to a named open milestone in BUILD.md, for example an oracle-binding status `owed-by: M<n>`.
- Gt treats the declared rows as owed, not missing, while the milestone is open. It reports them as owed on its checked line.
- Gt fails if:
  - the milestone is closed;
  - any owed row gains a test (the declaration is then stale and must be removed);
  - an owed id does not resolve;
  - any row outside the declaration lacks a test.
- The set is shrink-only, ratchet style. It can never grow without an explicit flag, like `baseline --grow`.
- Ga refuses to close a milestone that still owes rows.
- The skill and references document it. Migration: projects that used a local list can switch to it and delete that list.

## Acceptance
- Hard-TDD in this repo (locked RED first).
- Covers:
  - a design with owed rows passes;
  - a stale owed row fails;
  - an owed row on a closed milestone fails;
  - growing the set without the flag fails;
  - Ga refuses closure while rows are owed.
- Released as a machinery version that NIL adopts, deleting its local list.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
