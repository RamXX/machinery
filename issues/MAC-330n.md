---
id: MAC-330n
title: "Hook strict mode: block only lint errors in design sources the change touched"
status: open
priority: 3
type: feature
labels: [hook, from-0.11.0]
created_at: 2026-10-06T04:04:25Z
created_by: ramirosalas
updated_at: 2026-10-06T04:04:25Z
content_hash: "sha256:efc2afd351ce73ac4552cdf11b3d007af9d5cbe7c5e882a1f7af7d4982d1dc91"
---

## Description
0.11.0 B5 says hooks still block lint errors in the design sources the change touched. In strict mode the stop hook blocks every remaining design ERROR, including pre-existing ones in files the turn never touched, because hook state records only design/impl booleans. Record the touched design paths in the hook state (bounded) and, in strict mode, block only findings whose source path is touched (DRIFT and armed G4 keep blocking regardless). Acceptance: a strict stop over an untouched pre-existing lint error does not block; the same error in a touched file blocks; crash recovery and retention bounds still hold.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
