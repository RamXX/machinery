---
id: MAC-nkkg
title: "Decision: runtime_closure pins one platform; cross-arch reproduction is emulated, not native"
status: closed
priority: 4
type: decision
labels: [external-checkers, policy, from-next]
created_at: 2026-09-24T21:32:35Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:49Z
content_hash: "sha256:77cb74dad77d02e5bda25c4bf65e0401f970f34743e1d56a38ca1c2ac4157050"
closed_at: 2026-09-24T21:32:49Z
close_reason: "Recorded premise from NEXT-recovered.md (owner policy); no work. Reopen only if the policy changes."
---

## Description
Decision: checker runtime_closure pins one platform; cross-arch reproduction is emulated, not native evidence

Recorded policy: `checker.runtime_closure` pins one platform, so a design verified on linux/arm64 cannot be reproduced natively on an amd64 runner without re-pinning. The declared `--platform` is always used; a daemon with emulation (Rosetta, qemu/binfmt) reproduces the pinned userspace as emulated reproduction, explicitly not native-host test evidence (docs/external-checkers.md:439-441). A per-platform closure list (one digest per platform, each bound) remains an unapproved feature extension. Documented under MAC-gcrr AC2.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:32:49Z status: open -> closed

## Links


## Comments
