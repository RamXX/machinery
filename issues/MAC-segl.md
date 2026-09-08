---
id: MAC-segl
title: "Partial-witness journal recovery test slices past its own record"
status: closed
priority: 1
type: bug
labels: [formal, journal, recovery, ci]
created_at: 2026-09-08T23:26:30Z
created_by: ramirosalas
updated_at: 2026-09-08T23:41:59Z
content_hash: "sha256:0c4e4e19d8af5fc6897eaf5114331a4e0d42026804d14b29284c88d4981f45ad"
closed_at: 2026-09-08T23:41:42Z
---

## Description
TestFormalJournalRecoversEveryPartialWitnessByte fails intermittently (hosted macOS CI run 34288638843, new/byte-082; reproduced locally at new/byte-84 on 1 of 3 runs).

The cut loop bound comes from a probe record seeded in a different temp dir (internal/formal/journal_hardening_test.go:22), while each subtest appends record[:cut] from its OWN freshly seeded record (line 27). Native witnesses are formatted as unix:%x:%x:%x:%x over dev, inode, birthtime sec, birthtime nsec (internal/formal/durability_unix.go:31), so the encoded record length varies run to run with the hex width of the inode and the birthtime nanoseconds. Measured locally over 200 seeds: lengths 83 (3), 84 (51), 85 (146).

When the subtest record is at least 2 bytes shorter than the probe record, the top cuts slice past len(record) into the spare capacity of the json.Marshal buffer (cap 96 vs len 84), so the appended bytes are a complete witness record plus junk. The reader then correctly rejects the junk as malformed trailing data.

The reader is sound: every true prefix of a valid record is recovered. The defect is in the harness.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-08T23:41:42Z status: open -> closed

## Links


## Comments

### 2026-09-08T23:41:59Z ramirosalas
Fixed in 15b6389 on branch fix/journal (worktree /Users/ramirosalas/workspace/machinery-worktrees/fix-journal).

Root cause
- internal/formal/journal_hardening_test.go:22 (pre-fix) bounded the cut loop by len(probeRecord), a record seeded in a separate t.TempDir, while line 27 appended record[:cut] from the subtest's own seed.
- The environment value that moves the boundary is the hex width of the native witness fields formatted at internal/formal/durability_unix.go:31: fmt.Sprintf("unix:%x:%x:%x:%x", Dev, Ino, sec, nsec). Measured over 200 seeds on this host: record lengths 83 (3), 84 (51), 85 (146). TMPDIR path length is not the mechanism; the record carries only the fixed base name A.tla.
- When the subtest record was 2+ bytes shorter than the probe, record[:cut] read past len into the json.Marshal spare capacity (len 84, cap 96), so the journal received a complete witness record plus stray bytes. The reader parsed the complete line and then correctly rejected the stray bytes at internal/formal/transaction.go:917, 'formal transaction journal has malformed trailing data'.
- The reader is sound: every true prefix of a valid record is recovered. No production code changed.

Evidence
- Reproduced locally pre-fix at new/byte-084 in 1 of 3 runs of TestFormalJournalRecoversEveryPartialWitnessByte. CI hit new/byte-082, which is the last cut when the probe record is 83 bytes and the subtest record is 81.

Fix
- Each cut is bounded by the record it appends (forEachPartialFormalWitnessCut).
- New TestFormalJournalRecoversPartialWitnessAtPinnedWitnessWidths sweeps every cut with the creation-time fields pinned narrowest (0:0) and widest (7fffffffffffffff:7fffffffffffffff) through formalWitnessTimeCoarsener.
- New TestFormalJournalRejectsTrailingBytesAfterCompleteWitness pins the rejection side deterministically.

Verification, all green
- go test -race -count=1 ./internal/formal -run 'Journal|Transaction|Recover' -timeout 20m
- go test -count=1 ./internal/formal -timeout 25m
- 8 consecutive runs of the cut tests
- TMPDIR sweep at path lengths 65, 104, 184, 264
- gofmt clean, go vet clean, golangci-lint 0 issues
