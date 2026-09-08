---
id: MAC-csh4
title: "lint: unconvert flags the load-bearing Stat_t.Dev conversion on linux/amd64"
status: open
priority: 1
type: bug
labels: [ci, lint, portability]
created_at: 2026-09-08T12:44:36Z
created_by: ramirosalas
updated_at: 2026-09-08T12:46:29Z
content_hash: "sha256:5d208bf5ba7806f04e0ca643f41bb7665c828363abf519901ec05c802e555bad"
---

## Description
## Symptom

The hosted `lint` job (ubuntu-latest, golangci-lint v2.13.2, linux/amd64) fails on
the unreleased 0.7.0 tree at commit 06aed43. The local darwin/arm64 preflight run
of the same linter is clean, so the defect only appears on the first hosted run.

## Verbatim CI line

Run https://github.com/RamXX/machinery/actions/runs/34225004428 job `lint`:

```
internal/tdd/inode_unix.go:17:34: unnecessary conversion (unconvert)
	return inodeIdentity{dev: uint64(st.Dev), ino: st.Ino}, true
	                                ^
1 issues:
* unconvert: 1
```

## Cause

`internal/tdd/inode_unix.go` carries the `//go:build unix` constraint and is
compiled for every supported unix target. `syscall.Stat_t.Dev` does not have one
width across those targets: it is `int32` on darwin and `uint64` on linux/amd64.
The `uint64(st.Dev)` conversion is load-bearing on darwin and a no-op on
linux/amd64, so `unconvert` flags it there and only there. A single shared file
cannot satisfy both platforms with a bare expression.

## Expected

`golangci-lint run --config .golangci.yml ./internal/tdd/...` is clean on both
`GOOS=darwin` and `GOOS=linux` without disabling or suppressing `unconvert` for
the package, and `inodeKey` still produces a correct device/inode identity on
every supported unix target.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T12:46:29Z ramirosalas
Fixed on fix/ci-mirror at 08b698d.

internal/tdd/inode_unix.go:17 widened both Dev and Ino through a new type-parameterized helper widenIdentityField, so the conversion is never an identity conversion in any instantiation and needs no nolint.

Verification:
- GOOS=linux GOARCH=amd64 golangci-lint run --config .golangci.yml ./internal/tdd/... -> 0 issues (the same command on the pre-fix file reproduces the CI line verbatim).
- golangci-lint run --config .golangci.yml ./internal/tdd/... (darwin/arm64) -> 0 issues.
- go test -count=1 -run 'Inode|Widen|Hardlink' ./internal/tdd/ -> ok.
- GOOS=linux go build ./internal/tdd/ -> ok.
