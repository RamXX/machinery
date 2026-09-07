---
id: MAC-r0hy
title: "Ship windows/amd64 as a supported build target"
status: open
priority: 1
type: task
assignee: dev-win
parent: MAC-ui8a
created_at: 2026-09-07T21:43:48Z
created_by: ramirosalas
updated_at: 2026-09-07T21:57:52Z
content_hash: "sha256:db2c64ab5f438bac2c170c2a7bc62917e6c919eb06ddf88949a7841b0588d719"
labels: [accepted]
---

## Description
User requirement: release targets are exactly darwin (amd64/arm64), linux (amd64/arm64), windows (amd64). CI already has a windows/amd64 build entry; GOOS=windows build currently fails in internal/processscope (broker.go/scope.go reference unix-guarded symbols frameReader/newFrameReader/selfDigest/mustMarshal with no windows fallback). Fix: compile on windows with UNSUPPORTED_PLATFORM semantics per the accepted contracts (Open returns UNSUPPORTED_PLATFORM; custody/formal-JVM/assurance native lanes are unix-only features and stay honestly unsupported on windows; core CLI must build and basic non-custody commands remain functional). Add windows/amd64 to the release build matrix + archive naming + checksums. No windows native test execution is claimed.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-07T21:57:52Z ramirosalas
ACCEPTED 2026-09-07 — a8ff27d. Pure-Go helpers (frameReader/newFrameReader/selfDigest/mustMarshal) moved verbatim to unguarded frame.go; guardian_other.go gains 4-line honest UNSUPPORTED_PLATFORM frameReader.read stub; unix paths byte-identical (29/29 processscope, zero skips). windows amd64+arm64 full-repo builds; Open/Join/ServeInternal honestly refuse on windows; release.yml gains windows/amd64 (manifest 8->10 artifacts, checksums glob-based pick-up, actionlint clean); release-archive proven on a real windows binary; lane 8+8 green. WATCH (documented, not blocking): windows TEST-compile gaps in tdd/special_windows_test.go + gates attest_implementation_test.go (CI windows is build-only); install.sh refuses Windows (installer change out of scope — noted for future).
