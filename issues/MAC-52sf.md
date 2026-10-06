---
id: MAC-52sf
title: "Threat-driven TDD and shift-right verification (process change A1-A9, B1-B8)"
status: in_progress
priority: 1
type: feature
labels: [gates, process, security, delivered]
created_at: 2026-10-05T23:50:58Z
created_by: ramirosalas
updated_at: 2026-10-06T02:47:17Z
content_hash: "sha256:77d8c57e35ab8c7d5cb94977bee1c0d6b56d533f7b0ee2aefaa096ee08d1ee06"
assignee: ramirosalas
---

## Description
Encode adversary classification (design/threats.yaml, Gz-threat), threat invariants, paired accept/refuse criteria, RED threat table, fail-closed machine lint, no documented closure for security gaps, governed verifiers, threat-first review charter; and shift-right verification: per-change/landing/checkpoint cadence, checkpoint-only staleness never blocks hooks, locked-test amendment rule, parallelism, hook temp-state crash recovery. Migration: audit mode by default, enforce opt-in, ratchet for existing entities.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-10-06T02:47:15Z status: open -> in_progress
- 2026-10-06T02:47:15Z claimed by ramirosalas

## Links


## Comments

### 2026-10-06T02:47:15Z ramirosalas
Acceptance evidence (story/MAC-52sf at 4c04cc0b, release: prepare machinery 0.11.0):
- make preflight-fast green on the release commit (lint, vet, gofmt, docs gate, modelith renders, 8 example gate suites, golden corpus, gate-experiment suite).
- go test ./... (60m timeout): all packages pass except internal/install TestBootstrapReceiptCLI, which fails identically on unchanged main v0.10.3 (filed MAC-nvbd); TestInstallAndDoctorTargetAll failed once on a Java provisioning deadline under load and passes in isolation.
- E2E: examples/go-crm runs Gz in enforce mode, platform-green with --impl; real-binary probes refuse a missing negative test and a "documented" closure with the expected reasons; C-SESS-11/12 fail when the signature check is mutated away; --landing is green while a checkpoint run reports stale attestations.
- Consumer-corpus diff (0.10.3 vs candidate, five consumer designs): 0 new blocking, 0 new warnings, 0 resolved; audit notes only.
