---
id: MAC-y2l2
title: "Bug: governance hook-state grows unboundedly under heavy local test sweeps"
status: in_progress
priority: 1
type: bug
parent: MAC-ui8a
created_at: 2026-09-08T02:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T07:43:02Z
content_hash: "sha256:d4d8d56626d4e015cecb3d556ed2b10a5fdd54a3c939512d48f4b09bec244da8"
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
