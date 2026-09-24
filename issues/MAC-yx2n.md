---
id: MAC-yx2n
title: "Acceptance grammar diagnostics: point at the offending line (lint-acceptance or block scalars)"
status: open
priority: 3
type: feature
labels: [ga, accept, diagnostics, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:50Z
content_hash: "sha256:07d5e332001a727f6fdd9272f86c9d36caa0ed5bd38ddee74eee43955db64a58"
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


## History


## Links


## Comments
