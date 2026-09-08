---
id: MAC-2pid
title: "formal and nightly lane jobs install Go from go.mod and none of the pinned assurance runtimes"
status: open
priority: 1
type: bug
labels: [ci, integration-lane, mirror-drift]
created_at: 2026-09-08T17:26:40Z
created_by: ramirosalas
updated_at: 2026-09-08T17:26:40Z
content_hash: "sha256:402e27c1b4c7b441c96f071b1c6e385527b03aa3dd2302f91543453bf8f4464d"
---

## Description
## Symptom
Hosted formal run 34254790803 (commit f939081), integration-required job:

```
assurance provision: unsupported go runtime/pin "go version go1.27.0 linux/amd64" (pinned go1.27.1)
```

## Cause
.github/workflows/formal.yml and nightly.yml lane jobs set up Go with `go-version-file: go.mod` (the module floor, 1.27.0) and install none of Node 26.8.1, TypeScript 7.0.2, CPython 3.14.7, or Elixir 1.20.4/OTP 29.0.6; only ci.yml's lane job carried the exact pins. The lane verifies every identity before running any suite (fail closed by design), so those two jobs could never pass; they had simply never run before the first hosted push of the epic. The ordering fix 4160f60 (identity check before module warm-up) made the Go mismatch the first visible failure.

## Fix
One composite action (.github/actions/assurance-runtimes) owns the pins; the ci lane, ci test, ci native macOS, formal lane, and nightly lane jobs all use it.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
