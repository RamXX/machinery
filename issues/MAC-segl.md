---
id: MAC-segl
title: "Partial-witness journal recovery test slices past its own record"
status: open
priority: 1
type: bug
labels: [formal, journal, recovery, ci]
created_at: 2026-09-08T23:26:30Z
created_by: ramirosalas
updated_at: 2026-09-08T23:26:30Z
content_hash: "sha256:95086f75ed05a6368daf8af8c7e4e8ed25d8fdb470452ee74ffde12f7f313332"
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


## Links


## Comments
