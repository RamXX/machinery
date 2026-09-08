---
id: MAC-33sp
title: "Directory ABA witness blind on Linux 6.8 in containers: readRootDirectory accepts a create-delete ABA"
status: open
priority: 1
type: bug
labels: [security, linux, dirscan, gates]
created_at: 2026-09-08T16:46:10Z
created_by: ramirosalas
updated_at: 2026-09-08T17:29:58Z
content_hash: "sha256:8ea2aaf7f01a30b36ea7bdac73cda0be9cb110d23d57ddf9eb1a7c7afcf9a1f2"
---

## Description
## Symptom
`TestReadRootDirectoryRejectsCreateDeleteABA` (internal/gates/inventory_hardening_test.go:36) fails deterministically on Linux inside a Docker container, with the temp directory both on overlayfs and on an ext4 bind mount, as root and as uid 1000:

```
inventory_hardening_test.go:36: readRootDirectory accepted same-directory ABA: <nil>
```

Host: Ubuntu, kernel 6.8.0-138-generic x86_64, Docker 29.7.2, image golang:1.27.1, `--init --cpus=2`. Commit f939081 (0.7.0 candidate). The same test passes on the hosted ubuntu-latest runner (ci test job, run 34225004428 shows no gates failure) and on darwin.

## Why it matters
`readRootDirectory` (internal/gates/confinement.go:290) is the fail-closed inventory primitive: its witness must detect a create-then-delete between enumeration and use. If the witness is blind under a common Linux configuration (container, kernel 6.8, coarse timestamps) the ABA defense silently degrades to accepting the enumeration, which is the fail-open direction. Machinery will increasingly run inside containers (the Dagger migration, MAC-zafm).

## Ask
Determine which witness component is blind here (directory ctime granularity on 6.8 without multigrain timestamps, the mutation-event witness from MAC-o82q not being available or not being consulted, statx change cookie support, bind-mount semantics), reproduce on the VM (ssh root@5.78.87.91, workspace /root/machinery-ci only, do not disturb other containers), and either make the witness granularity-independent on this configuration or make readRootDirectory fail closed (refuse the enumeration with a diagnostic) when no reliable witness is available, never accept silently.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T17:29:58Z ramirosalas
## Blind component, with evidence

The blind component is the native change stamp itself: `dirscan.directoryChangeID` on Unix returns the inode ctime, and on Linux before multigrain timestamps (mainline 6.13) every inode timestamp comes from the kernel's coarse clock, one timer tick wide. There was no mutation-event conjunct in `internal/dirscan` or `internal/gates` at all; the MAC-o82q / MAC-3hzt sentinel exists only in `internal/formal` and `internal/runtimeclosure`, so the directory-inventory path never had a granularity-independent half to fall back to.

Reproduced on the VM (Ubuntu 24.04, kernel `6.8.0-138-generic`, `CONFIG_HZ=1000`, Docker 29, image `golang:1.27.1`), inside a container on overlayfs and on the ext4 bind mount, as root and as uid 1000. A statx/stat probe run in that container:

```
=== base /tmp ===                       (statfs type=0x794c7630 overlayfs)
statx mask=0x00001fff (asked 0x40001fff)
  STATX_CHANGE_COOKIE(0x40000000) present=false
  before   ctime=1788886182.858277178 mtime=...858277178 size=4096 nlink=2
  after    ctime=1788886182.858277178 mtime=...858277178 size=4096 nlink=2
  ctime identical across ABA: true   mtime identical: true   size identical: true
  inotify_add_watch(path): ok
  inotify_add_watch(/proc/self/fd): ok   proc-fd watch observed events: n=64
=== base /ext4tmp ===                   (statfs type=0xef53 ext4)
  ... identical result, ctime 1788886182.859277178 before and after
```

Three findings there:

1. The two directory ctimes taken 1 ms apart are `...858277178` and `...859277178`: identical sub-millisecond digits, exactly one tick apart. That is the coarse clock at HZ=1000, not a fine-grained timestamp. A create+delete+`utimensat` sequence completes inside one tick, so ctime, mtime, size and nlink all compare equal.
2. `STATX_CHANGE_COOKIE` is not exposed: a 0x40001fff request returns mask 0x1fff. Those kernels strip the bit from userspace requests, so `i_version` is not an available fallback stamp.
3. inotify is available and does observe the create/delete, both by path and through `/proc/self/fd/N`. So the mutation-event channel is a usable witness on exactly the configuration where the stamp is blind.

overlayfs vs the ext4 bind mount made no difference; neither did root vs uid 1000. The blindness is the clock, not the filesystem.

Why the hosted runner passes: `ubuntu-latest` (image `ubuntu24/20260831.293`) runs kernel **6.17.0-1022-azure**, which has multigrain timestamps (landed for 6.13). The kernel switches an inode to a fine-grained ctime after that ctime has been queried, so the ABA in the test lands on a strictly later stamp and the existing stat comparison catches it. The VM's 6.8 has no such path. The pass on CI is a property of the runner kernel, not of the code.

Scope of the fail-open was wider than the filed test. On the VM at f939081, six tests failed, all the same defect:

```
internal/gates   TestReadRootDirectoryRejectsCreateDeleteABA
internal/gates   TestReadRootDirectoryRejectsRenameABA
internal/gates   TestSourceInventoryRejectsRootRenameABA
internal/gates   TestSourceInventoryRejectsNestedAncestorRenameABA
internal/dirscan TestReadCatchesSameDirectoryCreateDeleteABA
internal/dirscan TestReadCatchesSameDirectoryRenameABA
```

plus, once those were fixed, `cmd/machinery` `TestReadOracleRootDirRejectsCreateDeleteABA` and `TestReadOracleRootDirRejectsRenameABA`, which carry their own copy of the same two-pass ctime witness.

## Fix

`af54c9f  fix(gates,dirscan): give directory inventories a clock-independent ABA witness`

`internal/dirscan` gains a `MutationChannel`: one kernel mutation-event channel (inotify on Linux, kqueue on darwin/BSD) with one cheap watch per directory, drained cumulatively and attributed per watch. Key properties:

- The watch is armed through the already-open directory descriptor (`/proc/self/fd/N` on Linux, a private `dup` on kqueue), never through a path, so an `os.Root` enumeration stays inside its confinement and the watch cannot be pointed at a different directory between open and watch.
- One channel per traversal, not per directory: the inotify instance is the scarce resource (`max_user_instances` default 128) while watches are cheap (`max_user_watches` ~60k). The source inventory, whose witnesses are revalidated at `Close` long after the walk, keeps its channel open for the whole custody window.
- A queue overflow or an unexpected drain error marks every watch mutated (fail closed). `Mutated` is sticky: once latched it never regresses.
- `IN_ATTRIB` / `NOTE_ATTRIB` are excluded so the inventory's own reads cannot be mistaken for a mutation.

Wired into `readRootDirectory` (witness-resident, so the post-subtree `revalidateRootDirectory` drains the same watch), `dirscan.readOnce`, `walkTreeDirBounded`, the embed-transaction inventory, the source-inventory custody handle, and `cmd/machinery` `readOracleRootDir`.

Where no channel can be armed the enumeration is refused, not accepted: `... has no reliable change witness: <reason>`. That covers an exhausted `max_user_instances`, a kernel without inotify, and any platform machinery does not ship for. Windows is deliberately unchanged and does not require a channel: its witness is NTFS ChangeTime read from the retained handle, which the coarse-clock blindness does not apply to.

## Tests added

Configuration-independent, so they reproduce the Linux blindness on any host including darwin:

- `TestReadRootDirectoryRejectsABAWithABlindChangeStamp`, `TestReadCatchesABAWithABlindChangeStamp`, `TestReadOracleRootDirRejectsABAWithABlindChangeStamp` pin the native change stamp to a constant, which is what a pre-6.13 kernel effectively does inside one tick, and require the ABA to be rejected anyway. Verified to fail on darwin when the mutation check is removed and pass with it.
- `TestReadRootDirectoryRefusesWithoutMutationWitness`, `TestDesignInventoryRefusesWithoutMutationChannel`, `TestReadRefusesWithoutMutationWitness`, `TestReadOracleRootDirRefusesWithoutMutationWitness` pin the fail-closed refusal and its diagnostic.
- `TestMutationWatchReportsRealNamespaceEvents` pins the channel itself against the real kernel.

## Verification

darwin (arm64): `go test -race -count=1 ./internal/gates ./internal/dirscan ./internal/hook -timeout 20m` -> ok (98.7s / 1.6s / 70.8s). gofmt clean, `go vet ./...` clean, `golangci-lint run` 0 issues for GOOS=darwin and GOOS=linux, GOOS=windows and GOOS=linux cross builds clean.

Linux 6.8 container (uid 1000, overlayfs):
```
ok internal/gates 41.1s  ok internal/dirscan 0.10s  ok internal/hook 22.7s  ok cmd/machinery 84.6s
ok internal/checker  internal/compose  internal/alloy  internal/tla  internal/refine  internal/ir  internal/lint
```
with `-race`: `ok internal/gates 84.4s / internal/dirscan 1.1s / internal/hook 60.4s`. Also green with `-race` on the ext4 bind mount (`TMPDIR=/ext4tmp`). A root run with `TMPDIR=/ext4tmp` shows three unrelated CLI-prerequisite failures caused by git dubious-ownership inside the container (`error obtaining VCS status`), not by this change.

## Follow-up worth a separate issue

The same coarse-ctime blindness applies to the file-level `*ChangeID` witnesses across the repo (`internal/install`, `internal/checker`, `internal/pack`, `internal/artifactset`, `internal/hook`, `internal/designlock`, `internal/modelithtx`, `cmd/release-archive`). Those compare a file's ctime plus size/mtime/identity and are blind to a same-size, same-content-length replace-and-restore inside one tick on the same kernels. Not touched here.

VM left clean: `/root/machinery-ci/repo` back on `candidate` at f939081 with no local branch and a clean tree; probe directory and both bundles removed; `ext4tmp` empty again; every container started for this work used `--rm` and none remain. No other container was touched.
