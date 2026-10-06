---
id: MAC-yx2n
title: "Acceptance grammar diagnostics: point at the offending line (lint-acceptance or block scalars)"
status: open
priority: 3
type: feature
labels: [ga, accept, diagnostics, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:acfccbb1f247e42d1a75aeca4f4474789fb3d5fe8d92b962d8e6a8253dd25cbb"
---

## Description
Acceptance file grammar: point at the offending line (machinery lint-acceptance) or accept block scalars

Problem (H2): the acceptance file grammar (single-quoted scalars; every id with its own prefix) cost each of three reviews one retry, and the YAML error names the sequence's first line rather than the offending apostrophe.

Proposed fix: `machinery lint-acceptance <file>` that reports the exact offending line and column with the rule broken (unescaped apostrophe, missing id prefix), and/or accept YAML block scalars for free-text fields so apostrophes need no escaping. Ga-accept error messages carry the same precise location.

Acceptance criteria:
1. An acceptance file with an unescaped apostrophe on line N reports line N (not the sequence start).
2. An id missing its prefix is reported with the id and line.
3. If block scalars are accepted, a note using `|` with apostrophes passes Ga-accept.
4. A valid file lints clean with exit 0.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Either add machinery lint-acceptance (exact line/column and rule, id-prefix diagnostics, exit 0 on clean) or map the YAML error to the offending line; add tests for ACs 1-4 including a block-scalar note. Evidence: Verified at HEAD 8c620d8f (v0.11.0). internal/gates/accept.go:614-624 parseAcceptance calls ir.LoadYAML (internal/ir/yaml.go:21) and reports 'invalid YAML: '+err verbatim, so the parser's sequence-start line is still surfaced; no lint-acceptance command exists (grep of cmd and docs finds none). Block scalars are already parseable by the YAML library but this is not documented or tested. Notes: Small, independent. Check whether the yaml error already carries the true line; the H2 report says it names the sequence start, unverified here.

## History


## Links


## Comments
