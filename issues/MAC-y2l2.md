---
id: MAC-y2l2
title: "Bug: governance hook-state grows unboundedly under heavy local test sweeps"
status: in_progress
priority: 1
type: bug
parent: MAC-ui8a
created_at: 2026-09-08T02:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T07:41:47Z
content_hash: "sha256:a0b3a77e2ed152d4eae01d01cda6a152bf6b8e31609301fea1f867e84d60f3d3"
assignee: dev-MAC-y2l2
follows: [MAC-3hzt, MAC-r0hy]
labels: [delivered]
---

## Description
Self-inflicted during release-candidate verification: go test -race ./... sweeps invoke the machinery CLI thousands of times; each hook invocation writes a route snapshot into ~/Library/Application Support/machinery-hook-state-<hash>/ with no retention. At 4096 entries the hook fails closed and bricks ALL shell/write tooling for every agent in the repo (including machinery doctor itself, which cannot remediate its own state). Remediation required a manual user prune. Needs: retention/compaction policy (e.g. keep newest N per route, size-bounded), and doctor must be able to repair its own state even at the limit. hwdb lineage. Evidence: session logs 2026-09-07, dir machinery-hook-state-288286948b3b49e6b991052d peaked >4096.

## Story Acceptance Criteria (derived from the description; recorded before implementation)

1. RETENTION POLICY, APPLIED ON EVERY WRITE.
   The per-user hook state store has a documented, constant retention policy enforced on
   every state-arming write:
   a. Per project root, at most a fixed number of route snapshots is retained (the newest
      by modification time); older ones are reclaimed. The snapshot just written is never
      reclaimed.
   b. The store as a whole has a retention ceiling strictly below the fail-closed entry
      limit (hookStateDirMaxEntries). Crossing it triggers compaction of reclaimable
      generations, oldest first, until the count is back at or under the ceiling.
   c. Total store bytes are bounded as a consequence: every retained file already has a
      per-file byte ceiling, so entries * per-file ceiling bounds the store.
   d. Both constants are documented in the source and in user-facing docs.
   e. Compaction is provably safe: a generation is reclaimable only when its ledger
      records a project root that no longer exists on disk. A generation whose root
      still exists, whose ledger is absent, unparseable, or carries no recorded root,
      or which holds crash evidence (durable temps or quarantines), is never reclaimed.
      Reclaiming uses the existing witness-checked quarantine deletion path under the
      project's own state lock; a contended generation is skipped, never forced.

2. SELF-REPAIR AT OR ABOVE THE LIMIT.
   a. The hook recovers from a store that is already at or above the fail-closed entry
      limit: a bounded inventory that fails on the entry limit triggers one compaction
      pass in repair mode (enumeration ceiling far above the fail-closed limit) and one
      retry, so a store bricked by an older version heals without a manual prune.
   b. If the store is still over the limit after compaction, the hook still fails closed,
      and its diagnostic names the store path, the counts, and the remediation command.
   c. `machinery doctor` reports the store path, entry count, retention ceiling and
      fail-closed limit without mutating anything, and does so even when the store is
      already at or above the fail-closed limit.
   d. `machinery doctor --repair` compacts the store using the same provably-safe policy,
      reports how many entries were reclaimed and how many remain, and works at or above
      the fail-closed limit.
   e. Neither doctor path creates the store directory when it does not exist, and neither
      disturbs the first-initialization / durable-loss-marker semantics.

3. TESTS (unit + integration, no mocks, no skips, isolated HOME so no test touches the
   user's live store).
   a. Negative proof of the old behavior: a store populated to the fail-closed entry
      limit makes the unbounded-growth read fail closed, and a governed hook event against
      such a store fails closed under the pre-fix inventory.
   b. Positive proof of recovery: the same over-limit store, with reclaimable generations,
      is compacted by the hook itself and by `machinery doctor --repair`, after which the
      governed event succeeds.
   c. Sweep proof: thousands of governed hook invocations against ephemeral project roots
      in an isolated temp store leave the store entry count at or under the retention
      ceiling, and always strictly below the fail-closed limit.
   d. Per-project route retention proof: many sessions against one root retain only the
      newest N route snapshots.

4. FAIL-CLOSED SEMANTICS PRESERVED (MAC-hwdb lineage).
   a. Every existing internal/hook fail-closed behavior is preserved: foreign or corrupt
      ledgers, noncanonical filenames, symlinks or special files in the store, durable
      crash temps, interrupted deletions, directory-identity and initialization-marker
      binding, and the routing-digest binding at stop time.
   b. Compaction never discharges a live obligation: it never removes a ledger for a root
      that exists, and it never clears design/impl obligation flags.
   c. The ledger format change that records the project root keeps the strict canonical
      parser: exactly one root line, in a fixed position, hex-encoded, and any other
      shape is corrupt and fails closed.
   d. The full internal/hook suite passes unchanged apart from tests that had to encode
      the new ledger line.

## Residual (documented, not hidden)

Reclamation treats an absent project root as a dead obligation. A project whose root is on
detached or unmounted storage at the moment the store is over its retention ceiling can
therefore have its obligation reclaimed; its next governed edit re-arms the obligation for
the whole tree. The alternative (retaining it) is what bricks every agent's tooling, so the
trade is stated rather than avoided.

## Acceptance Criteria


## Design


## Notes


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-08.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## History
- 2026-09-08T05:53:07Z status: open -> in_progress
- 2026-09-08T05:53:07Z auto-follows: linked to predecessor MAC-3hzt
- 2026-09-08T05:53:07Z claimed by dev-MAC-y2l2
- 2026-09-08T07:41:46Z status: in_progress -> in_progress
- 2026-09-08T07:41:46Z auto-follows: linked to predecessor MAC-r0hy

## Links
- Parent: [[MAC-ui8a]]
- Follows: [[MAC-3hzt]], [[MAC-r0hy]]

## Comments
