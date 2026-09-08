---
id: MAC-y2l2
title: "Bug: governance hook-state grows unboundedly under heavy local test sweeps"
status: in_progress
priority: 1
type: bug
parent: MAC-ui8a
created_at: 2026-09-08T02:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T15:22:20Z
content_hash: "sha256:9eebc29f415cb04f14c110c53dbd32ba3846478dab4c8b0461e186942023e8f5"
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
      limit, for ledgers this version wrote: a bounded inventory that fails on the entry
      limit triggers one compaction pass in repair mode (enumeration ceiling far above the
      fail-closed limit) and one retry. CORRECTED AT REVIEW: this does not extend to a
      store filled before the upgrade. A ledger written by an older version carries no
      project root, so nothing can tell its obligation from a live one and compaction
      retains every one of them. Such a store stays failed closed until a human removes
      its <digest>.state files (never the store directory), and that one-time remediation
      is documented in docs/claude-plugin.md and the CHANGELOG rather than implied away.
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

### 2026-09-08T07:43:02Z ramirosalas
DELIVERED on branch story/MAC-y2l2 (from epic tip 4bc4dad, not merged, not pushed).

Commits:
- 2fd8180 test(hook): keep the package's own governance state out of the user's store
- 8f04b97 fix(hook): bound the governance state store and repair it in place
- 5feffbd test(hook): run the socket-marker case instead of skipping it
- 74ff6b4 test(hook): prove a doctor report never materializes the store

HEAD / proof SHA: 74ff6b4f18422fd2c8a1731c09341baa52a66d01

## ROOT CAUSE (reproduced, not inferred)

Two independent causes, both fixed.

1. The store had no retention. Every project root that armed an obligation and
   never discharged it left a ledger plus a route snapshot forever. Nothing
   could tell a dead obligation from a live one, because the store is keyed by
   a sha256 of the project root and that digest cannot be inverted.
2. internal/hook's own suite wrote into the invoking user's real store. It had
   no TestMain, so os.UserHomeDir/os.UserConfigDir resolved to the real home.
   Measured on the epic tip at 4bc4dad, with HOME pointed at an empty temp dir:
     go test -count=1 ./internal/hook  ->  28 entries left in the store
   Every one is an obligation for a t.TempDir() root that no longer exists, and
   no Stop event can ever discharge it. Roughly 150 package runs reach 4096.

## WHAT CHANGED

internal/hook/retention.go (new, 736 lines)
- hookStateRouteRetention = 8: route snapshots kept per project root.
- hookStateDirRetentionCeiling = 512 (hookStateDirMaxEntries/8): store-wide
  ceiling checked on every arming write; hookStateDirRetentionTarget = 256 is
  what one pass compacts back to.
- Bytes are bounded as a consequence: every retained file already has its own
  byte ceiling (hookStateMaxBytes, hookRouteMaxBytes).
- compactHookStateDir reclaims whole generations, oldest ledger first, and only
  ones it can prove are dead. Screens without locking (a store of live roots
  costs 23 ms, measured, not one lock per generation), then removes in batches
  of 64 under a store-wide lock.
- Self-repair: an inventory that hits the 4096-entry limit compacts once under a
  repair-mode ceiling and retries, so a store bricked by an older version heals
  without a manual prune. Still over the limit afterwards: fails closed with a
  diagnostic naming the store, the counts, and `machinery doctor --repair`.
- StateReport(w, repair) is the doctor surface; it never creates the store.

internal/hook/hook.go
- The ledger records its canonical project root as one hex-encoded `root` line
  immediately after `revision`. This is the whole basis of safe reclamation.
  The parser is as strict as the rest of the format: exactly one root line, in
  that position, lowercase hex of an absolute cleaned path; anything else is
  corrupt and fails closed.
- Both arming paths publish the route snapshot and then apply both bounds.
- Project state locks are tracked per process, so reclamation never takes a lock
  an arming event in the same process is standing on (POSIX record locks belong
  to the process; a second acquisition would replace it and its release would
  drop the event's protection).
- Four store enumerations now go through readHookStateDir.

internal/dirscan/dirscan.go: ErrTooManyEntries sentinel (message unchanged).
cmd/machinery: `machinery doctor` reports the store; `machinery doctor --repair`
compacts it. Both work at or above the fail-closed limit.
docs/claude-plugin.md + README.md: the policy, both constants, both residuals.

## PROOF

Commands, all run at 74ff6b4 in the story worktree:

1. go test -race -count=1 -v ./internal/hook ./internal/dirscan
   ok internal/hook 293s, ok internal/dirscan 1.6s
   139 top-level PASS, 196 subtests PASS, 0 FAIL, 0 SKIP.

2. go test -count=1 -coverprofile ./internal/hook ./internal/dirscan
   internal/hook 77.6% of statements, internal/dirscan 74.6%, total 77.4%.
   (internal/hook was 77.3% before the last two test commits; the package has
   never been at 80%. retention.go's own uncovered statements are IO-failure
   branches.)

3. go test -race -count=1 -v -run 'TestDoctor|TestHookCmd|TestFlagUsage|TestRepositoryContract|TestPreflight|TestCmdTest|TestConcurrentCmdTest|TestInstall|TestUninstall|TestUpdate' ./cmd/machinery
   ok 219s, 31 top-level PASS, 10 subtests, 0 FAIL, 0 SKIP.

4. Blast radius (every dirscan consumer), go test -race -count=1:
   ok internal/gates 92s, internal/checker 98s, internal/lint, internal/ir,
   internal/compose, internal/refine, internal/tla, internal/alloy. 0 FAIL.

5. golangci-lint run --config .golangci.yml ./internal/hook/... ./internal/dirscan/... ./cmd/machinery/...
   0 issues. gofmt clean.

6. pvg verify (authored files, --include-tests): PASSED (7 files, 0 issues).
   pvg gates: PASS (161 warn, 2 skipped) - the warnings are the repo's existing
   file_loc warnings; retention.go (697) and retention_test.go (422) join that
   pre-existing list.

Not run, deliberately: scripts/preflight.sh, make preflight, go test ./... and
go test -race ./... across the module. The epic reserves the full sweep for the
final gate, and this host is measuring another agent's integration lane. Skipped
packages have no code path overlap with hook state, dirscan error wrapping, or
the doctor command.

Every test run above used an isolated HOME. The user's live store at
~/Library/Application Support/machinery-hook-state-288286948b3b49e6b991052d held
153 entries before this work and holds 153 now; nothing in it was pruned.

### Acceptance criteria

AC 1a per-root route bound            TestRouteSnapshotsAreBoundedPerProjectRoot
                                      (20 sessions on one root, never more than 8 snapshots, the
                                      just-written one always survives, ledger obligation unchanged)
AC 1b store ceiling on every write    TestHookStateStoreStaysBoundedAcrossThousandsOfInvocations
                                      (2000 real governed events on ephemeral roots: peak <= 512,
                                      never within 3500 of the fail-closed limit; 117s)
AC 1c bytes bounded                   by construction, stated in retention.go and docs
AC 1d constants documented            retention.go, docs/claude-plugin.md, README.md
AC 1e reclamation provably safe       TestCompactionNeverReclaimsLiveCorruptOrCrashEvidenceGenerations
                                      (live root, legacy ledger with no root, corrupt ledger, ledger
                                      with a durable temp: all retained; exactly the one dead
                                      generation reclaimed, and the corrupt one is reported)
                                      TestCompactionNeverReclaimsAGenerationThisProcessHasLocked
                                      TestStoreFullOfLiveObligationsStaysIntactAndStillArms
AC 2a hook self-repairs               TestStoreAtTheEntryLimitFailsClosedAndThenSelfRepairs
AC 2b still fails closed, actionable  same test: with the repair attempt spent (the state every hook
                                      process was permanently in before this fix), the inventory fails
                                      with the entry limit, names the store and the remediation, and
                                      the governed event is denied
AC 2c doctor reports, no mutation     TestDoctorReportsAndRepairsTheGovernanceHookStateStore,
                                      TestStateReportStatesTheStoreAndRepairsItAboveTheLimit
                                      (entry count identical before and after the report)
AC 2d doctor --repair at the limit    same two tests (4100+ entries -> under the limit)
AC 2e never creates the store         TestStateReportNeverCreatesTheStore (store and initialization
                                      marker both still absent after report and repair)
AC 3a negative, old behavior          TestStoreAtTheEntryLimitFailsClosedAndThenSelfRepairs asserts
                                      dirscan.Read at the real limit fails with ErrTooManyEntries
AC 3b positive, recovery              same test: fresh process, event allowed, store under the ceiling,
                                      and the obligation it just armed still present
AC 3c thousands of invocations        2000 events, isolated temp store, see AC 1b
AC 3d per-route retention             see AC 1a
AC 4a fail-closed semantics kept      full internal/hook suite green with 0 skips, including the
                                      foreign/corrupt-state, quarantine, crash-temp, directory-binding
                                      and routing-digest cases
AC 4b never discharges a live one     AC 1e tests, plus the live obligation surviving self-repair
AC 4c strict ledger parser            TestHookStateLedgerRejectsEveryNoncanonicalProjectRootLine
                                      (9 shapes: two root lines, root after design, root after route,
                                      non-hex, uppercase, empty, relative, uncleaned, root with no
                                      touch class), TestHookStateLedgerBindsItsCanonicalProjectRoot
AC 4d existing suite unchanged        no existing hook test was weakened; two were changed and both
                                      got stronger (see below)

### Changes to existing tests, disclosed

- cmd/machinery/diag_test.go: two doctorRunUnlockedTo call sites take the new
  repair argument. No assertion changed.
- internal/hook/hook_test.go: the "config special file" case bound a Unix-domain
  socket directly at the marker path. A socket address is limited to about 104
  bytes and every temp dir this suite creates exceeds it, so the case skipped
  itself on every run, before my change as well (proved by running it with the
  original system TMPDIR). It now binds under a short base, renames the socket
  into place, and FAILS rather than skips when the socket cannot be created.
  The package now has 0 skipped tests.

## RESIDUALS AND JUDGEMENT CALLS FOR THE PM

1. Reclamation treats an absent project root as a dead obligation. A root on
   detached or unmounted storage is indistinguishable from a deleted one, so its
   obligation can be reclaimed while the store is over its ceiling; the next
   governed edit in that project re-arms the whole-tree obligation. Documented
   in the code and in docs/claude-plugin.md. The alternative is what bricked
   every agent's tooling.
2. Reclamation of a dead generation is witness-checked (two agreeing reads, then
   removal through a retained os.Root authority, then proof of absence) but not
   quarantined, unlike every other deletion in this file. A quarantine exists so
   an interrupted deletion of a live obligation stays restorable; a dead
   generation has nothing to restore. Measured: quarantined removal costs 63 ms
   per generation on this host, so repairing a 4096-entry store would have taken
   137 s with every tool blocked; witnessed removal costs 0.4 ms and the same
   repair takes about 1.5 s. If the PM wants the quarantine back, it is a
   one-function change and a slower repair.
3. Ledger format: a machinery older than this one reads a ledger written by this
   one as noncanonical and fails closed. A downgrade therefore blocks until the
   affected <digest>.state files are removed (not the directory, whose marker is
   the durable-loss evidence). Documented in docs/claude-plugin.md.
4. The sweep test costs 117 s (2000 real governed events; about 75 ms each is the
   pre-existing fsync cost of one hook event on this host, measured with and
   without the new bound and unchanged by it). It is the honest cost of the proof
   the story asked for. Say so if the epic gate wants it smaller; 600 events
   still crosses the ceiling three times.
5. Concurrency: reclamation excludes concurrent enumeration through a store-wide
   lock, proved across two real processes by
   TestCompactionExcludesConcurrentEnumerationInAnotherProcess. I found this the
   hard way: without the lock, a compaction burst exhausted dirscan's 8 retries
   in a concurrent sweep and DENIED an unrelated governed event. In-process
   goroutines cannot model this (POSIX record locks belong to the process), which
   is why that test uses a child process.

## NOT DONE, ON PURPOSE

No push, no merge, no epic branch or integrate worktree touched, no
pvg story merge, no pvg loop setup, no full preflight.

## LEARNINGS

- The bug had a second, larger half the report did not name: the product's own
  test suite was the thing filling the user's store, at 28 entries per package
  run, because internal/hook had no TestMain while cmd/machinery had a thorough
  one. When a package writes to user-scoped paths, the sandbox belongs in
  TestMain, not in a helper each test has to remember to call. 28 of the 115
  tests called the helper.
- Retention on a fail-closed store is a safety change disguised as housekeeping.
  The only defensible policy was one that can PROVE an obligation is dead, which
  needed the ledger to record its project root; the store's digest key cannot be
  inverted. Everything else (LRU, age) discards live obligations silently.
- Durable-deletion cost dominates any bulk operation: 63 ms per file with the
  quarantine dance, 0.4 ms witnessed. Measure before designing a budget; I nearly
  added a work budget for a problem that was really per-file fsync.
- Bulk mutation of a shared directory breaks other processes' bounded
  enumeration retries. Ordinary single-file writes do not (1000 concurrent
  arming events, zero failures); a burst of removals does. Exclusion has to be
  scoped to the burst, or every governed event on the machine pays for it.
- POSIX record locks belong to the process, not the acquisition. Two consequences
  bit me: an in-process concurrency test proves nothing about locking, and
  reclamation had to track which project locks this process already holds or it
  would have released an arming event's own lock out from under it.

### 2026-09-08T09:39:22Z ramirosalas
# Code review: MAC-y2l2 (story/MAC-y2l2, tip 74ff6b4, base 4bc4dad)

Read-only review. No tests, builds, or `go` commands were run; no file in the repo was modified.
Evidence is file:line against the branch tip unless stated otherwise.

Verdict up front: **ACCEPT WITH CHANGES**. The mechanism is well built and the fail-closed
discipline is preserved. Two findings are material: the fix does not remediate the store the bug
report was actually filed about, and route retention holds the store-wide lock for an unbatched
quarantined-deletion loop that can exceed the 10 s lock-wait budget of unrelated governed events.

---

## 1. Correctness of the retention policy

**Verdict: sound in mechanism, but the reclamation predicate does not cover the reported
incident.**

The bound itself is right. `boundHookStateStore` (retention.go:196) runs on the arming write path,
reads the store, and compacts only when `len(entries) > hookStateDirRetentionCeiling`
(retention.go:207). Compaction targets 256 (retention.go:49), so the post-write count is always at
or under 512, an eighth of the 4096 fail-closed limit (hook.go:71). Bytes follow from entries by
construction, as documented (retention.go:28-33). The constants are documented in source and in
docs/claude-plugin.md:196-201. AC 1a-1d hold.

Reclamation safety (AC 1e) is genuinely conservative:

- Only a generation whose ledger parses and whose recorded root is provably absent is reclaimable
  (`hookStateGenerationLooksDead` retention.go:373, `hookStateRootVanished` retention.go:472).
- `hookStateRootVanished` accepts **only** `os.ErrNotExist` (retention.go:477). EACCES on the root
  or its parent, ESTALE on a dead NFS mount, EIO, ELOOP all fall through to "not vanished" and the
  obligation is retained. A governed repo whose root is temporarily unreadable through permissions
  or a broken network mount is therefore safe.
- A generation with no ledger, a corrupt ledger, a legacy ledger with no root, or any durable temp
  or noncanonical entry bound to its base is retained (`groupHookStateGenerations`
  retention.go:344-350, `hookStateEntryEvidence` retention.go:296, the `generation.blocked ||
  !generation.ledger` guard at retention.go:263).
- Ordering is oldest-ledger-first (retention.go:352-358), so a freshly armed obligation is the last
  candidate a bounded pass would reach. That materially shrinks the residual window below what the
  story's residual note claims.

**The residual that is real.** An unmounted volume path (`/Volumes/X` after eject) does not exist,
so it reads as dead. The documented mitigation, "the next governed edit re-arms the obligation for
the whole tree", does not cover the ordering that matters: edits happen, the volume is ejected, other
repos push the store past 512, the volume is remounted, and the session's Stop then runs with no
obligation and no gate. That is a fail-open, not a re-arm. It is narrow (the root must vanish while a
session is live and the store must be over its ceiling at that moment) and the oldest-first ordering
makes it narrower still. Accepting it is defensible; the residual text in docs/claude-plugin.md:214
and retention.go:466-471 should say "the gate can be skipped for that session", not only "the next
edit re-arms", because the current wording understates it.

**The finding that blocks AC 2a as written.** Reclamation requires the `root` line, and only this
version writes it (hook.go:3455). A ledger written by 0.6.11 has no root line, so `record.root == ""`,
so `hookStateRootVanished("")` returns false at retention.go:474 and the generation is retained
forever. The tests confirm this is intended: `TestCompactionNeverReclaimsLiveCorruptOrCrashEvidence
Generations` fabricates `"revision 1\ndesign\n"` and asserts the "legacy ledger without a recorded
root" survives (retention_test.go:253, :281).

Consequence: **a store actually bricked by an older version does not heal.** Every one of its 4096
entries is a rootless ledger for a deleted temp root, and none is reclaimable. Self-repair compacts
zero, `doctor --repair` reclaims zero, and the machine stays bricked until a manual prune, which is
exactly the remediation this story exists to remove. The user's own live store (153 entries, written
by 0.6.11) is in that state today. A rootless ledger for a *live* project does get its root line back
on the next governed event in that project (hook.go:3489 `if root != ""`), but a rootless ledger for a
dead root is never rewritten, so precisely the garbage is permanently unreclaimable.

Every proof of AC 2a and AC 3a uses new-format fixtures: `writeDeadGenerations` writes
`hookStateRootLine(deadRoot)` (retention_test.go:68), and the doctor test replicates a ledger produced
by the new binary (diag_test.go:~525). So the negative test is real for the format this version
writes, and untested for the format the incident produced.

The docs assert the healed case outright: "A store that is already at or above the fail-closed limit,
left by an older version, repairs itself" (docs/claude-plugin.md:207). That statement is false as
written and it sits on the mission-critical path, where a user reading it will not run the manual
prune that they still need.

**Required change (docs, minimum):** correct docs/claude-plugin.md:207 and the AC 2a claim to say
self-repair applies to ledgers this version wrote, and give the explicit one-time remediation for a
pre-upgrade store (remove `<digest>.state` files, never the directory, which the doc already says
correctly at :219).

**Owner ruling wanted (code, optional):** whether `doctor --repair` should get an explicit,
non-default mode that discards rootless ledgers. It is a deliberate fail-open (each affected project
re-arms on its next governed edit) and therefore not something to add silently, but without it the
0.6.11 -> 0.7.x upgrade path still requires a manual prune.

## 2. Witness-checked but not quarantined reclamation

**Verdict: the trade is justified; the evidence loss is nil for the class of file being removed.**

`removeWitnessedHookStateFile` (retention.go:437) reads twice, compares the witness, removes through
the retained `os.Root`, and proves absence afterwards. It is materially the same integrity check as
the quarantined path minus restorability.

What a quarantine buys is a restorable intermediate state for an *interrupted* deletion. A generation
that reaches this path has been proved dead twice, the second time under the project's own state lock
(retention.go:400-431), and the removal is routes-first, ledger-last (retention.go:419-430), so an
interrupted pass leaves the obligation readable and reclaimable again. There is nothing an audit could
recover from a quarantine of a ledger whose only content is a touch class plus a path that no longer
exists. A completed malicious or mistaken deletion is unrecoverable with or without quarantine, so the
quarantine does not defend the adversarial case either.

The 137 s vs 1.5 s figure justifies the one-time repair. It also justifies the steady state, which the
delivery does not spell out: a routine 512 -> 256 pass removes ~256 files, which is ~16 s quarantined
against ~0.1 s witnessed, and 16 s is already past the 10 s `hookStateLockWaitLimit` that every other
governed event waits on. So the quarantine was not viable on the write path at all, not just at the
4096 limit. That is a stronger argument than the one made and it should be the one in the comment at
retention.go:391-397.

One gap worth stating: nothing anywhere records *which* roots were reclaimed. `StateReport` gives
counts (retention.go:591), the write path is silent (correct for a hook). Acceptable for 0.7.1; a
future `doctor --repair --verbose` naming reclaimed roots would close it.

## 3. Ledger format change

**Verdict: a real one-way compatibility break, correctly disclosed, with a failure mode that is a
brick until the user deletes files.**

- New binary reads an old ledger: fine. No `root` line means `record.root == ""` (hook.go:3562 case
  is simply not taken), the record parses, governance is unaffected, and only reclamation is refused.
- Old binary reads a new ledger: `parseHookStateRecord` at 4bc4dad has no `root` case, so the line
  falls to `default:` and the ledger is "corrupt or noncanonical". Every read of that project's state
  then fails, and the hook fails closed. That is a **brick for that project**, not a warning: shell
  and write tools are denied until the `<digest>.state` file is removed.

Mixed-binary exposure is real but bounded by the store being per-user-home. The two cases that matter:

- **CI pinned at 0.6.11 and a workstation at 0.7.x** with no shared home: no exposure. Different
  stores, no interaction. This is the common case.
- **Two checkouts sharing one per-user state dir** with different binaries (a `~/.local/bin`
  machinery at 0.7.x and a repo-local or plugin-cached 0.6.11): full exposure. Whichever project the
  new binary arms becomes unreadable to the old one. The plugin-cache version skew guard
  (README.md:655) catches the plugin path, not a hand-placed older binary.
- **Downgrade** (`machinery update --version v0.6.11` after running 0.7.x): every project armed by
  0.7.x is bricked for that user until the files are removed.

The break is stated in docs/claude-plugin.md:216-220 with the correct remediation and the correct
warning not to remove the directory. That is the right disclosure. The parser is as strict as claimed:
exactly one root line, position-bound by `classIndex != 0 || routeStarted || pendingStarted ||
record.root != ""` (hook.go:3563), lowercase hex of a nonempty absolute cleaned path
(hook.go:3466-3476), and nine noncanonical shapes are covered by
`TestHookStateLedgerRejectsEveryNoncanonicalProjectRootLine` (retention_test.go:100). AC 4c holds.

**Required change:** the downgrade break is user-visible and currently lives only in
docs/claude-plugin.md. It belongs in the CHANGELOG `[Unreleased]` section as a compatibility note.
See finding 8.

## 4. Self-repair at 4096 and `doctor --repair`

**Verdict: the original bug (doctor cannot remediate its own state) is fixed for new-format
ledgers. No path deletes the marker or the store and then fails closed.**

- `compactHookStateDir` enumerates with `dirscan.Read(dir, hookStateDirRepairMaxEntries)` directly
  (retention.go:230), bypassing the 4096 ceiling. It never goes through the bounded reader, so it
  cannot fail on the state it exists to repair.
- Nested inventories inside a compaction (interrupted-deletion recovery at retention.go:412) are
  covered by `hookStateRepairDepth` (retention.go:69), which raises the ceiling
  (`hookStateDirEntryCeiling` retention.go:172) and suppresses the shared-lock reacquisition
  (retention.go:150). Both are necessary and both are correct.
- `countHookStateEntries` (retention.go:645) also reads under the repair ceiling, so a plain
  `doctor` report states the real count of an over-limit store instead of failing on it. AC 2c holds
  and `TestStateReportStatesTheStoreAndRepairsItAboveTheLimit` proves the entry count is identical
  before and after a non-repair report (retention_test.go:~330).
- `StateReport` returns before touching anything when the store is absent (retention.go:557), so
  neither path creates the store or the marker. `TestStateReportNeverCreatesTheStore` asserts both
  (retention_test.go:~470). AC 2e holds.
- No path removes the marker or the store directory. Compaction removes only `<digest>.state` and
  `<digest>.state.route-<digest>.json` entries it has proved dead; `.store-identity`
  (hook.go:2021) classifies as `hookStateEntryUnrelated` (retention.go:283) and is never a candidate.

**One dependency worth stating.** `compactHookStateDir` calls `validatedStateDirectoryBinding()`
(retention.go:242), which requires a present, non-legacy initialization marker (hook.go:2532-2536).
If the marker is missing or the directory identity has drifted, `doctor --repair` reports ERROR and
cannot compact. In that state the hook already fails closed for the same reason, so repair could not
have helped, but the diagnostic will say "could not be compacted" rather than naming the marker
problem. Worth one line in the report or the doc so a user is not left chasing retention when the
real fault is the binding.

**Minor:** `StateReport` never returns a non-nil error on any path (every branch prints and returns
`nil`), so the error branch in `reportHookStateStore` (diag.go:355) is unreachable and untested.
Either drop the error return or make the resolve failure at retention.go:551 return it.

## 5. Locking

**Verdict: no deadlock or ordering hazard found in production paths. One latent recursion hazard,
one inaccurate comment, and one lock-hold-time regression (see finding 6 of this section).**

Lock order is consistent:

- Arming write: project state lock (held by caller) -> store shared (`readHookStateDirLocked`
  retention.go:145) -> store exclusive (`retainProjectRouteSnapshots` retention.go:527, and
  `flush` retention.go:257) -> marker lock (via `readBoundedHookStateFile` hook.go:2518).
- Compaction: marker lock taken and released inside `validatedStateDirectoryBinding` *before* the
  store lock (retention.go:242 vs :257), then store exclusive, then per-project locks
  **non-blocking** (`filelock.Acquire` retention.go:406, contended -> skip at :408).

Because compaction never *waits* on a project lock, the ABBA cycle that project->store and
store->project would otherwise create cannot close. I checked the counterpart directions
specifically: `ensureStateDir` holds the marker lock (hook.go:2171) and does not enumerate the store
(it calls `ensureStateDirectoryIdentity`, `readStateInitializationMarker`,
`writeStateInitializationMarker`, none of which reach `dirscan`), so nothing takes the store lock
while holding the marker lock. `removeHookFileWitness` and `readBoundedHookRootFile` do not
enumerate either, so `retainProjectRouteSnapshots` holding the store exclusive lock never
re-enters the shared acquisition.

TOCTOU between screening and removal is closed correctly. `hookStateGenerationLooksDead` is an
unlocked screen (retention.go:373) and the comment says so; `reclaimHookStateGeneration` re-reads
the ledger and re-proves root absence *under the project's own state lock*
(retention.go:414-418), which is the same scope `acquireStateLock` takes (hook.go:2960). An absent
root cannot be canonicalized, so no other process can be arming that generation while the lock is
held. This is right.

**Per-process lock tracking (retention.go:88-116, hook.go:2977-2986):** correct and proved by
`TestCompactionNeverReclaimsAGenerationThisProcessHasLocked` (retention_test.go:~415), including
the negative half (releasing the lock makes the same generation reclaimable). Scope strings match:
`statePath` builds `filepath.Join(dir, stateFileName(absRoot))` (hook.go:1897-1915) and
`generation.base` is `<digest>.state` under the same `dir`, so `ledgerPath + ".session"` is
byte-identical to the arming scope.

**Comment inaccuracy on a mission-critical path.** Three comments justify the tracking with "POSIX
record locks belong to the process, not to the acquisition: taking this scope again here would
replace the lock" (retention.go:401-405, hook.go:2977, retention_test.go:~407). That is true only
on the `fcntl` build (`//go:build aix || (solaris && !illumos)`, fcntl_unix.go:1). darwin, linux and
every other unix machinery supports use `flock` (flock_unix.go:1), where a second acquisition on a
new fd conflicts and returns contended, and the `fcntl` build additionally has an in-process
reservation map (fcntl_unix.go:23-32) that refuses the second acquisition outright. So the stated
hazard does not exist on the platforms this ships to, and on the platform where record locks do
apply it is already prevented one layer down. Keep the mechanism (it is cheap and it makes the
invariant explicit), but fix the comments: they justify a guard on a premise that is false where it
runs.

**Latent recursion.** If `readHookStateDir` were ever to return `ErrTooManyEntries` under the
1,048,576-entry repair ceiling while a compaction `flush` holds the store exclusive lock (via
`recoverHookDeletionQuarantines` at retention.go:412), the self-repair branch would call
`compactHookStateDir` recursively and its own `flush` would block on a store lock this process
already holds, for 10 s, then fail. Unreachable in practice (>1M entries), but a one-line guard
(`if hookStateRepairDepth.Load() > 0 { return nil, hookStateOverLimit(...) }` at the top of the
self-repair branch, retention.go:129) removes the class entirely.

**Lock-hold-time regression, and the one change I would require in this file.**
`retainProjectRouteSnapshots` takes the store-wide **exclusive** lock (retention.go:527) and then
runs an unbatched loop of *quarantined* removals (retention.go:534-543 calling
`removeHookFileWitness`, hook.go:3794). Compaction explicitly batches at 64 for exactly this reason
and says so (retention.go:57-60, :250-253). Route retention does not. At the delivery's own measured
63 ms per quarantined removal, ~158 surplus snapshots exceed the 10 s `hookStateLockWaitLimit` that
`readHookStateDirLocked` waits on (retention.go:151), at which point governed events in **unrelated
repositories** fail closed and deny their tool call. Surplus was unbounded before this change, so a
long-lived project on an existing store can well be carrying hundreds of route snapshots, and the
first governed event after the upgrade pays all of it at once.

Required: bound the per-write surplus removal the same way compaction bounds a batch, either by
releasing and reacquiring the store lock every N files or by capping how many surplus snapshots one
write reclaims (the bound is per-project and converges over a few events either way).

**Also worth stating:** the arming write path now performs two additional whole-store enumerations
per governed event, `routeStatePaths` inside retain (retention.go:485 -> hook.go:1939 ->
`readHookStateDir`) and `boundHookStateStore`'s own read (retention.go:203). At 512 entries this is
cheap and the delivery's per-event timing is unchanged, but it should be in the docs as a stated cost
of the bound.

## 6. Tests

**Verdict: substantive, with three gaps.**

What is real:

- **Negative before / positive after.** `TestStoreAtTheEntryLimitFailsClosedAndThenSelfRepairs`
  (retention_test.go:~190) asserts `dirscan.Read(dir, hookStateDirMaxEntries)` returns
  `ErrTooManyEntries` at the limit, then, with the process's single repair attempt deliberately
  spent via `beginHookStateAutoCompaction()`, asserts a real governed event returns
  `"permissionDecision":"deny"` and that the diagnostic names the store and
  `machinery doctor --repair`. It then resets the flag to simulate a fresh hook process and asserts
  the same event is allowed and the just-armed live obligation survives. That is a genuine
  before/after, not a shape assertion.
- **The bound is asserted, not just exercised.**
  `TestHookStateStoreStaysBoundedAcrossThousandsOfInvocations` (retention_test.go:~155) runs 2000
  real governed events against ephemeral roots and fails on `peak > hookStateDirRetentionCeiling`
  and on `count >= hookStateDirMaxEntries`. The bound is the assertion.
- **The fail-open direction is tested.** `TestStoreFullOfLiveObligationsStaysIntactAndStillArms`
  (retention_test.go:~440) fills the store with obligations for a root that *exists*, and asserts
  nothing is reclaimed and governance still works. This is the test that matters most and it is
  present.
- **Cross-process exclusion.** `TestCompactionExcludesConcurrentEnumerationInAnotherProcess`
  (retention_test.go:~355) re-executes the test binary and asserts the parent's enumerations never
  fail while the child compacts 1200 generations. Real subprocess, real contention.

Sandbox: **yes, on every path I could find.** `internal/hook/TestMain` (testmain_test.go:26)
redirects HOME, USERPROFILE, all five XDG vars, APPDATA/LOCALAPPDATA and TMPDIR/TMP/TEMP into one
private root before any test runs, and honors an inherited root so the compaction child shares its
parent's store (testmain_test.go:29-33). `TestHookTestStateStaysInsideItsOwnSandbox`
(testmain_test.go:~97) is a standing guard that asserts both the resolved store path and the
initialization marker path are inside that root, which covers the darwin case where
`os.UserConfigDir` ignores XDG and derives from HOME. `cmd/machinery/TestMain`
(cmd/machinery/testmain_test.go:28) already did the same, and the new doctor test additionally sets
HOME/XDG_CONFIG_HOME per-test. One caveat: `cmd/machinery/TestMain` deliberately skips the
HOME/TMPDIR redirect when the binary is re-invoked as a checker fixture
(cmd/machinery/testmain_test.go:87-89); no doctor path runs in that mode, and `StateReport` never
creates the store, so the worst case there is a read of the real store, not a write.

Gaps:

1. **The negative test does not cover the format the incident produced.** Every over-limit fixture
   is written with `hookStateRootLine` (retention_test.go:68). There is no test that a store of
   *rootless* 0.6.11 ledgers is reclaimed, because it is not. Add a test that pins the actual
   behavior (a store of rootless dead ledgers is retained and the hook still fails closed), so the
   limitation is encoded rather than implied. That test will also prevent someone later "fixing" it
   silently into a fail-open.
2. **The 2000-invocation test samples the count 1 in 25** (`if i%25 != 0 && i != invocations-1`,
   retention_test.go:~176). The bound is enforced after every write so the sampled peak is sound,
   but the sampling is not what makes it sound and the test does not say so.
3. **Route retention's downstream effect is untested.** `TestRouteSnapshotsAreBoundedPerProjectRoot`
   proves at most 8 snapshots survive and the just-written one always does, but no test runs a Stop
   for a session whose snapshot was pruned. That session now falls into
   `loadRouteSnapshot`'s recovery branch (hook.go:2899-2927), which blocks on divergent configs
   where it previously matched exactly. Identical configs recover identically, so the risk is small,
   but it is a new fail-closed path introduced by an 8-snapshot policy and it should have one test.

The two changed existing tests are justified and both got stronger, as claimed. The
`diag_test.go` change is a mechanical argument addition with no assertion changed. The socket-marker
case (hook_test.go:3434-3451) was genuinely skipping on every run because a Unix-domain socket
address is capped near 104 bytes; binding under a short base and renaming into place makes the case
execute, and `t.Fatalf` replaces `t.Skipf` on the bind. Two notes: the case can *still* skip if the
rename fails (hook_test.go:~3449), so "0 skipped tests" is conditional, not structural; and
`shortSocketBase = "/tmp"` (shortsocket_unix_test.go:8) hardcodes a path outside the sandbox this
same story introduced. It is test-only and cleaned up, but it is the one place in the branch that
contradicts its own thesis, and it will fail on a host with a non-writable `/tmp`. Deriving it from
the pre-sandbox `os.TempDir()` captured in `TestMain` would be consistent.

Coverage: 77.6% for `internal/hook` is below the 80% floor. The developer discloses that the package
has never met it and that retention.go's uncovered statements are IO-failure branches. Not introduced
here, but 736 lines of new mission-critical code went in without a per-file coverage figure; worth
asking for retention.go's own number.

## 7. House style and hygiene

**Verdict: clean.**

- No em dashes anywhere in the diff (0 occurrences across all added and removed lines).
- No emoji.
- No "honest", "truthful", "transparent" self-labeling in code, comments, docs, or commit messages.
  (The word "honest" appears once in the nd delivery comment, "the honest cost of the proof", which
  is conversation with the PM, not generated content. Still better dropped from the habit.)
- No TODO/FIXME/XXX. The only `fmt.Fprintln(os.Stderr, ...)` calls are `TestMain` bootstrap
  diagnostics and the compaction child's failure reporting, which are correct uses.
- Commit messages are substantive and explain cause before change. Each carries the required
  `Claude-Session:` trailer.
- Comments are accurate with the one exception in finding 5 (the POSIX record lock premise) and
  the one in finding 1 (the residual understates the Stop-time consequence).
- Naming and structure follow the file's existing conventions. `retention.go` at 697 lines joins the
  repo's pre-existing `file_loc` warning list rather than introducing a new class of warning.

## 8. Merge risk onto current main (5c35463: 24ac158 + 5c35463)

Branch files that main also changed since 4bc4dad:

| File | Conflict |
| --- | --- |
| `cmd/machinery/diag.go` | **Yes, textual.** Both add a helper immediately before the `// reportCheckerBinaries` comment: main adds `reportSkillRelease` (main's diag.go:352-378), the branch adds `reportHookStateStore` (branch diag.go:352-364). Same insertion anchor. Resolution is trivial: keep both. Additionally the branch changes the `doctorRunTo`/`doctorRunUnlockedTo` signatures (diag.go:245, :269) while main edits the body at :285 and :321; those are different lines and should auto-merge, but the whole function needs re-reading after resolution. |
| `cmd/machinery/diag_test.go` | **Yes, textual, twice.** Both append a new test at EOF after `copyDoctorFixture`, and both add an import (branch: `fmt`; main: `machversion`). Both resolutions are keep-both. |
| `README.md` | **No.** The branch edits the "### Prerequisites" paragraph at ~455-460; main's edits start at 484. Disjoint hunks. |

**Semantic conflicts: none.** Main's change makes doctor fail on a stale installed skill; the branch
makes doctor fail on an over-limit hook state store. They add independent entries to the same
`failures` slice and compose correctly. `internal/gates`, `internal/install`, `internal/processscope`
and the examples/golden churn on main have no code-path overlap with hook state, dirscan error
wrapping, or the doctor command.

**Behavior change to flag for the release, not a conflict.** `machinery doctor` now exits nonzero
whenever the hook state store is at or above 4096, or whenever the store path cannot be resolved or
inspected (`reportHookStateCount` returns false at retention.go:610; `StateReport` returns false at
retention.go:553, :562, :566). The "above the ceiling but below the limit" case correctly returns
true (retention.go:615), so the common state does not newly fail. But any CI or Makefile target that
treats `machinery doctor` as a gate now has a new failure source, and a container with no resolvable
HOME will now fail doctor where it may have passed. That belongs in the CHANGELOG.

**Missing release artifact.** `CHANGELOG.md` has an `[Unreleased]` section (CHANGELOG.md:6) and the
branch adds nothing to it, despite shipping a new user-visible flag (`doctor --repair`), a doctor
exit-status change, and a one-way ledger format break with a manual remediation. Required before
this goes into a release.

---

## Verdict

**ACCEPT WITH CHANGES.**

Required before merge:

1. **Correct the self-repair claim.** docs/claude-plugin.md:207 states that a store left over-limit
   by an older version repairs itself. It does not: pre-0.7.x ledgers carry no `root` line and are
   permanently unreclaimable (retention.go:474). Rewrite that paragraph to scope self-repair to
   ledgers this version wrote, and state the one-time manual remediation for a pre-upgrade store
   (remove `<digest>.state` files, keep the directory). Same correction to the AC 2a wording in the
   story.
2. **Batch or cap route-retention surplus removal.** `retainProjectRouteSnapshots` (retention.go:527)
   holds the store-wide exclusive lock across an unbatched loop of 63 ms quarantined removals. Past
   ~158 surplus snapshots it exceeds the 10 s lock-wait budget and denies governed tool calls in
   unrelated repositories. Apply the same batching rationale compaction already states at
   retention.go:57-60.
3. **Fix the record-lock comments.** retention.go:401-405, hook.go:2977 and retention_test.go:~407
   justify the per-process lock tracking with `fcntl` semantics that do not apply on darwin or linux
   (flock_unix.go:1). Keep the guard, restate the reason.
4. **Strengthen the residual wording.** retention.go:466-471 and docs/claude-plugin.md:214 say the
   next governed edit re-arms the obligation. Add that if no edit follows before Stop, that session's
   gate does not run.
5. **Add a CHANGELOG `[Unreleased]` entry** covering `doctor --repair`, the doctor exit-status
   change, and the ledger downgrade break with its remediation.
6. **Add the missing negative test** pinning that a store of rootless legacy ledgers is retained and
   still fails closed, so the limitation cannot be silently converted into a fail-open later.

Recommended, not blocking: guard the recursive-compaction path (retention.go:129); derive
`shortSocketBase` from the pre-sandbox temp dir instead of hardcoding `/tmp`; add one test for a Stop
whose route snapshot was pruned by the 8-snapshot bound; drop or wire up `StateReport`'s dead error
return.

**Release placement: 0.7.1, with the six changes above applied.** No owner design ruling is needed to
ship it. The mechanism is correct, the fail-closed semantics are preserved, and the one deliberate
fail-open (absent root means dead) is narrow, ordered last by oldest-first reclamation, and
documented. The single question that does want an owner ruling is whether `doctor --repair` should
gain an explicit, non-default mode that discards rootless legacy ledgers. That is a real fail-open
decision and it should not be made by a developer or a reviewer; it can also wait, because with
change 1 applied the user has a correct manual remediation for the only store that needs it.

### 2026-09-08T10:15:28Z ramirosalas
REWORK for the ACCEPT WITH CHANGES review, targeted at 0.7.1.

Rebased onto main (5c35463) first. New tip: c9322b990da0e2f3fc641f40d3c8687924b049c8
Branch story/MAC-y2l2, four rebased commits plus one rework commit. Nothing pushed, nothing merged.

  c9322b9 fix(hook): bound the route-retention lock hold and state what self-repair covers
  12238a8 test(hook): prove a doctor report never materializes the store
  9702ca1 test(hook): run the socket-marker case instead of skipping it
  06dc163 fix(hook): bound the governance state store and repair it in place
  4fd0cb3 test(hook): keep the package's own governance state out of the user's store
  5c35463 release: date 0.7.0 to its cut day   (main)

## Rebase

Both predicted conflicts appeared and both were keep-both, exactly as the review said.

- cmd/machinery/diag.go: main's `reportSkillRelease` and this branch's `reportHookStateStore` both
  insert immediately before the `// reportCheckerBinaries` comment. Both kept, in that order
  (diag.go:359 and diag.go:388). The `doctorRunTo` / `doctorRunUnlockedTo` signature change merged
  with main's body edits without conflict; I re-read the whole function after resolving, and main's
  stale-skill failure and this branch's store failure append independently to the same `failures`
  slice (diag.go:319-330 and diag.go:353).
- cmd/machinery/diag_test.go: both append a test at EOF and both add an import. Both kept; the
  import block carries `fmt` and `machversion` together.
- README.md auto-merged, disjoint hunks, as predicted.

## The six required changes

1. **Self-repair claim corrected.** docs/claude-plugin.md:215-227 now reads "Self-repair covers
   ledgers this version wrote, and only those", states that a pre-upgrade ledger carries no project
   root so compaction retains every one of them, and gives the one-time remediation (remove the
   `<digest>.state` files, never the directory) in the same paragraph as the claim rather than only
   in the downgrade note below it. The doctor exit-status change is stated there too. AC 2a in the
   story is corrected in the same terms and marked CORRECTED AT REVIEW.

2. **Route-retention lock hold bounded.** internal/hook/retention.go:630 `retainProjectRouteSnapshots`
   now caps one write at `hookStateRouteReclaimBudget = 16` surplus snapshots (retention.go:71) and
   removes them through `removeRouteSnapshotBatch` (retention.go:692) in runs of
   `hookStateRouteReclaimBatch = 8` (retention.go:77), acquiring and releasing the store-wide lock
   per run. Worst case per lock hold is now about half a second at the delivery's measured 63 ms per
   quarantined removal, against a 10 s lock-wait budget; worst case per governed event is about one
   second, and a backlogged project converges over the next few events. Pinned by
   TestRouteRetentionReclaimsABoundedShareOfSurplusPerWrite (retention_test.go:603), which asserts
   both halves: one write reclaims no more than the budget, and repeated writes converge to 8.

3. **Record-lock comments corrected.** retention.go:508-516 now says what is true: flock builds
   (darwin, linux, the other BSDs) scope the lock to the descriptor so a second open conflicts and
   reports contention, and the fcntl builds (AIX, Solaris), where a record lock does belong to the
   process, hold an in-process reservation that refuses it one layer down. The guard stays, and the
   comment says why it stays: it states the invariant where the decision is made and keeps the
   accounting truthful (retained deliberately, not incidentally). Same correction at
   retention_test.go:409-411. hook.go:2977 `trackedStateLock` carries no platform claim.
   retention.go:170-175 additionally records the thing that actually made the in-process concurrency
   test misleading: `hookStateRepairDepth` is process-wide, which is exact for a hook process
   handling one event and wrong for goroutines, which is why exclusion is proved with a child
   process.

4. **Residual wording strengthened.** retention.go:602-611 and docs/claude-plugin.md:229-236 now say
   it is a fail-open, not a deferral: the next governed edit re-arms the obligation, but if the
   volume returns and the session ends with no edit before Stop, that session's gate does not run at
   all. Both keep the narrowing facts (the root must vanish while a session is live and the store
   must be over its ceiling, and oldest-first reclamation reaches a freshly armed obligation last).

5. **CHANGELOG `[Unreleased]` entry added** (CHANGELOG.md:7-50): Added for `doctor --repair` and the
   retention policy with both constants; Fixed for hook self-repair with its explicit pre-upgrade
   limitation and remediation; Changed for the doctor exit-status change including the no-resolvable-
   HOME case and the note that a store above the ceiling but below the limit still passes;
   Compatibility for the ledger downgrade break in both directions with its remediation.

6. **Negative test for the pre-upgrade format added.**
   TestStoreOfLegacyRootlessLedgersIsRetainedAndKeepsFailingClosed (retention_test.go:519) fabricates
   4104 rootless `revision 1\ndesign\n` ledgers, then asserts: compaction reclaims 0 and retains at
   least 4096; `doctor --repair` reports not-ok and still names the 4096-entry limit; the store's
   size is unchanged; a real governed event is denied; and the denial changes nothing. The
   limitation is now encoded, so converting it into a fail-open takes a deliberate edit that breaks
   this test.

## The recommended items

- **Recursive compaction guarded.** retention.go:149 the self-repair branch now returns the
  over-limit error when `hookStateRepairDepth > 0`, so a nested inventory inside a compaction can
  never start a second pass that would block on the store lock its own caller holds.
- **shortSocketBase derived, not hardcoded.** The two `shortsocket_*_test.go` files are gone; the
  helper now lives beside the sandbox it depends on (testmain_test.go) and returns
  `hookTestPreSandboxTempDir`, captured in TestMain before TMPDIR is redirected
  (testmain_test.go:19-24, :43). No build tags: the body is platform-neutral and the case still
  skips on Windows for its own reason. The socket case still executes and passes.
- **StateReport's dead error wired.** retention.go:727 returns the resolve failure instead of
  printing and returning nil, so `reportHookStateStore`'s error branch (diag.go:395) is reachable.
- **Pruned-snapshot Stop tested** (was recommended, added):
  TestStopRecoversRoutingAfterRetentionPrunedItsSnapshot (retention_test.go:569) arms 12 sessions on
  one root, asserts the oldest snapshot was reclaimed, then asserts `loadRouteSnapshot` recovers the
  same configuration through the shared-route branch and that a real Stop for that session does not
  block on routing.
- **Two review notes taken as one-liners.** The repair error now says a missing or drifted store
  initialization marker fails there and fails every governed event for the same reason, so the
  message is read as being about identity rather than retention (retention.go:735). The docs state
  the bound's cost: two whole-store enumerations per arming event (docs/claude-plugin.md:212-214).

Not done, as instructed: no repair mode that discards rootless ledgers. That stays an owner ruling.

## PROOF at c9322b9

1. go test -race -count=1 -v ./internal/hook ./internal/dirscan
   ok internal/hook 220s, ok internal/dirscan 1.3s
   142 top-level PASS, 196 subtests PASS, 0 FAIL, 0 SKIP.
   (Was 139/196 before the rework; the four new tests are the difference.)

2. go test -race -count=1 -v -run 'TestDoctor|TestHookCmd|TestFlagUsage|TestAssuranceDocs|TestRepositoryContract|TestPreflight|TestCmdTest|TestConcurrentCmdTest' ./cmd/machinery
   ok 67s, 39 top-level PASS, 0 FAIL, 0 SKIP. TestAssuranceDocs* covers the CHANGELOG, including
   TestAssuranceDocsStandaloneMachineryOnly, so the new entry carries no delivery-process identity.

3. go test -count=1 -coverprofile ./internal/hook
   internal/hook 77.7% of statements (was 77.6%).
   retention.go specifically, from the same profile: 78.9% of statements (270 of 342), mean 85.2%
   across its 25 functions. The uncovered statements are IO-failure branches (Lstat/Open/Remove
   errors inside the removal and enumeration paths) and the exhausted-bound early return.

4. golangci-lint run --config .golangci.yml ./internal/hook/... ./internal/dirscan/... ./cmd/machinery/...
   0 issues. gofmt clean. go vet clean.

5. pvg verify (authored files, --include-tests): PASSED (7 files, 0 issues). The thin-file finding on
   the Windows socket stub is gone with the file.
   pvg gates: PASS (161 warn, 2 skipped), same pre-existing file_loc warnings.

Not run, per the constraints: any full sweep. Another agent is load-testing this host.

Every run used the sandboxed HOME. One observation worth recording rather than acting on: the user's
live store went from 153 entries when this story started to 203 now, from this session's own governed
tool calls across worktrees, not from any test. Every one of those entries is pre-upgrade format, so
they are exactly the population that change 1 documents as needing the one-time manual removal.
Nothing in that directory was pruned.

## LEARNINGS

- The review caught the failure mode I had inverted: I proved self-repair against fixtures my own
  code writes, and the incident's store is written by the version before it. A negative test is only
  negative for the format it feeds in; when a change adds a field that gates behavior, the fixture
  without the field is the test that matters.
- I justified a guard with lock semantics that are true on two platforms this does not ship to. The
  guard was right and the reason was wrong, which is worse than an unexplained guard, because the
  next reader inherits the wrong model. The actual reason the in-process concurrency test was
  misleading was a process-wide flag of my own, not the lock flavor.
- I batched the compaction path for lock-hold time and then left the route path, which does the same
  kind of bulk removal on the same shared store, unbatched. The rule needed to be attached to the
  operation (bulk mutation of a shared directory) rather than to the function I happened to be
  writing at the time.

### 2026-09-08T15:22:19Z ramirosalas
Owner ruling 2026-09-08: manual remediation is fine for pre-upgrade (rootless) ledgers; do NOT add an automatic doctor --repair mode that discards them. Story stays parked for 0.7.1 with the reviewed rework at c9322b9.
