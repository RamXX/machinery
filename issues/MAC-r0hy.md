---
id: MAC-r0hy
title: "Ship windows/amd64 as a supported build target"
status: closed
priority: 1
type: task
assignee: dev-win
parent: MAC-ui8a
created_at: 2026-09-07T21:43:48Z
created_by: ramirosalas
updated_at: 2026-09-07T21:57:52Z
content_hash: "sha256:e79a600cc4863c4c1b257b6eb6103e421ad04a0a356e96ea22c1e90a1ee077e0"
labels: [accepted]
closed_at: 2026-09-07T21:57:52Z
close_reason: "Accepted: windows/amd64 ships as build target with honest unsupported-custody semantics"
---

## Description
User requirement: release targets are exactly darwin (amd64/arm64), linux (amd64/arm64), windows (amd64). CI already has a windows/amd64 build entry; GOOS=windows build currently fails in internal/processscope (broker.go/scope.go reference unix-guarded symbols frameReader/newFrameReader/selfDigest/mustMarshal with no windows fallback). Fix: compile on windows with UNSUPPORTED_PLATFORM semantics per the accepted contracts (Open returns UNSUPPORTED_PLATFORM; custody/formal-JVM/assurance native lanes are unix-only features and stay honestly unsupported on windows; core CLI must build and basic non-custody commands remain functional). Add windows/amd64 to the release build matrix + archive naming + checksums. No windows native test execution is claimed.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-07T21:57:52Z status: open -> closed

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-07T21:57:52Z ramirosalas
ACCEPTED 2026-09-07 — a8ff27d. Pure-Go helpers (frameReader/newFrameReader/selfDigest/mustMarshal) moved verbatim to unguarded frame.go; guardian_other.go gains 4-line honest UNSUPPORTED_PLATFORM frameReader.read stub; unix paths byte-identical (29/29 processscope, zero skips). windows amd64+arm64 full-repo builds; Open/Join/ServeInternal honestly refuse on windows; release.yml gains windows/amd64 (manifest 8->10 artifacts, checksums glob-based pick-up, actionlint clean); release-archive proven on a real windows binary; lane 8+8 green. WATCH (documented, not blocking): windows TEST-compile gaps in tdd/special_windows_test.go + gates attest_implementation_test.go (CI windows is build-only); install.sh refuses Windows (installer change out of scope — noted for future).
