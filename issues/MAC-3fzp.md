---
id: MAC-3fzp
title: "Effect carriers under --impl: each declared CARRIES{} target resolves to a code site"
status: open
priority: 2
type: feature
labels: [gy, carriers, consistency-layer, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:e85d150e18d3029aa9c2e351967cf6317dbd02a3db9aefbe109ba275d09db7d1"
---

## Description
Effect carriers under --impl: each declared CARRIES{} target resolves to a code site

Problem (H2, M3): machines declared effects in named units (`holdBatchWatermark` answers a held watermark, `recordParseMarker` answers markers, the Failing arm answers an alert, `completeRun` answers a sealed manifest), and the persistence driver carried a closed list of "report keys" that silently dropped anything not a column. Four declared effects went nowhere for weeks; a fifth was added to the closed list to silence the raise that would have reported it. All gates were green; a review found it by reading.

Delivered: the design-side half shipped in 0.10.0. `CARRIES{}` (column, outbox, sink, signal, action) is parsed by Gx-trace and `rules/consistency/carriers.dl` raises `finding_effect_uncarried` and `finding_carrier_misplaced` in Gy-rules (commits f31c0752, 68fff1be). That catches the four "no carrier named" H2 cases.

Residual (not delivered): the entry's `--impl` half. Nothing checks that a named carrier resolves to an implementation site, so the fifth H2 case (a carrier named that nothing implements, or implemented by a closed list that drops it) still passes.

Proposed fix: under `machinery check --impl`, resolve each `CARRIES{kind:target}` to a code site the way Gt resolves an oracle id (whole-token match in production sources scoped by the Architecture Contract component that owns the unit, with a per-kind resolver: a column in a migration or schema, an outbox emission, a sink or signal name, another machine's action). Emit `carrier_unimplemented` (name TBD) as a Gy or Gt finding with the unit and carrier.

Acceptance criteria:
1. A synthetic design plus impl where a unit declares `CARRIES{column:held_watermark}` and no production source names it fails under `--impl` with a finding naming the unit and carrier.
2. The same case passes once the column appears in the owning component's sources.
3. Without `--impl` the finding never fires.
4. Resolution uses the same whole-token and comment-stripping rules as Gt (no credit from comments or disabled files).
5. Bundled examples stay green or are corrected with the change recorded in CHANGELOG.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Under --impl resolve each CARRIES{kind:target} to a code site with Gt's whole-token, comment-stripped rules scoped by the owning component, per-kind resolvers, and emit a finding; tests and examples. Evidence: Design-side half shipped in 0.10.0 (rules/consistency/carriers.dl, finding_effect_uncarried and finding_carrier_misplaced, CHANGELOG lines 411-415). grep for carrier_unimplemented across repo returns nothing: no --impl resolution of CARRIES targets to code sites. Notes: Per-kind resolvers (column in migration, outbox emission, sink, signal, action) are the bulk of the work; consider starting with column and action kinds.

## History


## Links


## Comments
