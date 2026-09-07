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
updated_at: 2026-09-07T21:43:48Z
content_hash: "sha256:341b7aab34736ac1c5b1cc3d8a802122c9b41f17c620d6143a9513e22e2caa0c"
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
