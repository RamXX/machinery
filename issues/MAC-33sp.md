---
id: MAC-33sp
title: "Directory ABA witness blind on Linux 6.8 in containers: readRootDirectory accepts a create-delete ABA"
status: closed
priority: 1
type: bug
labels: [security, linux, dirscan, gates]
created_at: 2026-09-08T16:46:10Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:05Z
content_hash: "sha256:d89be6c690c74f131625942ce0e4259c36a4d3754e60f8c7ad6e103a24a5226d"
closed_at: 2026-09-24T21:32:05Z
close_reason: "Core fix shipped in 0.8.0 (a56bbe32, 1f62e9ab). Residuals filed as MAC-n35t. Triage 2026-09-24."
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
- 2026-09-24T21:32:05Z status: open -> closed

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

### 2026-09-08T17:40:44Z ramirosalas
Review 2026-09-08 (read-only): HOLD FOR 0.7.1. Reasons: dirscan.WalkBounded's cross-subtree window (dirscan.go:280 to :317) stays stamp-only and is the walker behind the hook's design inventory; the refusal is an unretried deny on every agent action with no sysctl-naming diagnostic (EMFILE/ENOSPC should retry, only ENOSYS should refuse); no watch/descriptor release test, overflow path untested, refusal pinned at 3 of 6 sites; frozen-stamp tests unguarded on Windows; Compatibility text claims enumeration is unchanged and omits max_user_watches, /proc availability, and the darwin per-directory descriptor hold; walkPluginCacheTopology in internal/install has the same directory blindness. Branch fix/aba-linux (af54c9f) stays parked. Full review: /private/tmp/claude-501/-Users-ramirosalas-workspace-machinery/3d758f9b-7707-4e06-8a3a-75759a3c3946/scratchpad/review-aba.md (copied below).

# Review: af54c9f, directory ABA witness (fix/aba-linux), against the 0.7.0 cut

Read-only review. Base f939081. Line references are at af54c9f (the working tree is at
main/497419a, so all citations below were taken from `git show af54c9f:<path>`).

The diagnosis in MAC-33sp is sound and the evidence is complete: coarse ctime at CONFIG_HZ=1000,
STATX_CHANGE_COOKIE stripped, inotify visible on the same configuration, and the hosted runner
passing only because it runs 6.17. Nothing below disputes the defect or the choice of remedy. The
question is whether this particular implementation belongs in a release cut today.

---

## 1. Operational risk of the refusal on the hot path

### Instances and watches opened

Per governed PreToolUse the hook makes at minimum four sequential `dirscan.Read` calls on the
per-user state directory, and 12 to 24 more when the tool's cwd sits below the project root
(`durableProjectStatePresent` runs its four-inventory loop twice per unmarked ancestor,
internal/hook/hook.go:2432-2470). Each `dirscan.Read` now opens its own inotify instance:
`readOnce` calls `watchDirectory(dir)` at internal/dirscan/dirscan.go:107, which is
`NewMutationChannel` + `Watch` (internal/dirscan/mutation.go:145-155). So one hook invocation
performs 4 to 28 `inotify_init1`/`inotify_add_watch`/`close` cycles, sequentially, holding at most
one instance at a time.

`machinery check` is different in shape. `validateDesignInventoryBounded`
(internal/gates/confinement.go:259), `walkTreeDirBounded` (:476), `embedResiduePathsBounded`
(internal/gates/embedtransaction.go:1313) and `walkSourceFilesBounded`
(internal/gates/imports.go:136) each open one channel for a whole traversal and add one watch per
directory via `readRootDirectory` (:343). The imports gate runs `goModules`, `tsPackages` and
`manifestDependenciesWithWorkspace` (each its own `walkTreeDirBounded`) before
`walkSourceFilesBounded`, so those are sequential, not nested. Gates run with no goroutines
(no `go func`/errgroup in internal/gates). Peak concurrent instances per machinery process is
therefore 1, occasionally 2. Peak concurrent *watches* is the number of directories in the tree,
held for the whole traversal, bounded only by `designInventoryMaxEntries = 100_000`
(internal/gates/confinement.go:20) and `implementationDirectoryMaxEntries = 100_000`
(internal/gates/imports.go:18). Watches are never removed mid-traversal; the source inventory holds
them for the entire custody window by design (internal/gates/imports.go:60-64).

So the two exhaustion surfaces are not the same one:

- `fs.inotify.max_user_instances` (default 128, per real uid): pressure is proportional to the
  number of concurrent machinery *processes*, not to tree size. N agents plus worktrees would need
  roughly 128 processes inside the same microsecond window. Low probability on a laptop. It is
  materially higher under the Dagger migration (MAC-zafm): containers on one host that share a uid
  share one counter unless each gets its own user namespace.
- `fs.inotify.max_user_watches` (8192 on Debian-family defaults, 65536 on Ubuntu): pressure is
  proportional to directory count per traversal, per concurrent agent, and it competes with the
  editors and file watchers the same user is already running. A developer whose VS Code already
  sits near the ceiling plus a `machinery check` over a few thousand directories is a realistic
  ENOSPC. This is the likelier of the two, and it is the one nothing in the change or the CHANGELOG
  anticipates.

On darwin and the BSDs there is a third: `mutation_bsd.go:39` dups the caller's descriptor per
watch and holds it in `c.held` until `close()` (:79-86). One traversal of a 5,000-directory tree
holds 5,000 file descriptors for its whole duration. Survivable under Go's raised RLIMIT_NOFILE,
but it is an unbounded-by-design accumulation with no test and no ceiling of its own.

### Does the refusal deny the tool call

Yes, and it is indistinguishable at the decision layer from a policy violation. Inventory errors
reach `deny(...)` at internal/hook/hook.go:886, :946, :880, :892, or return early as
`PermissionDecision: "deny"` at :680-684. Errors that escape as Go errors exit non-zero, and
hooks/machinery-hook.sh:68-71 converts any non-zero exit into `exit 2` (blocked). There is no
infrastructure-versus-policy classification anywhere in internal/hook; the only fail-open is the
narrow `unavailableDurableProjectState()` at hook.go:2474-2477 for an unresolvable HOME. This is
exactly the failure class MAC-y2l2 addressed from the other side (c9322b9: "deny tool calls in
unrelated repositories").

### Retry and diagnostic

Neither. `readOnce` returns the witness refusal as a plain error, not wrapped in `ErrChanged`, so
`Read`'s bounded retry loop returns immediately (internal/dirscan/dirscan.go:69-72). The new test
pins that on purpose ("a missing witness is not a transient change", dirscan_test.go). That is
right for a kernel that has no inotify and wrong for EMFILE/ENOSPC, which are textbook transients:
the eight-attempt loop with backoff already exists a few lines away and is not used.

The diagnostic names nothing actionable. `newMutationChannel` wraps the raw errno
(mutation_linux.go:26-32), so the user sees `open directory mutation channel: too many open files`
or, for a watch, `watch directory for mutation events: no space left on device`. The second will
send people to check disk space. Neither sysctl appears in the runtime message; only the CHANGELOG
names `max_user_instances`, and it never names `max_user_watches`.

**Net:** a transient EMFILE or ENOSPC becomes a hard deny of an agent's tool call, unretried, with
a diagnostic that does not say why. The probability is low on the hook path and moderate on the
gates path, and the blast radius is every tool call in every governed repo for that user.

---

## 2. Correctness

**Arming window.** Clean. In all three sites the watch is armed before the first enumeration and
before the first stat: dirscan.go:107 precedes `captureDirectoryState` at :112; confinement.go:343
precedes `inventoryChangeID` at :348 and `readRootDirEntries`; oracle.go:801 precedes
`oracleChangeID` at :806 and the first `readOracleDir`. A mutation before arming is harmless
(the first read simply observes the later state); the property that matters is that the watch spans
both passes, and it does. No re-read after arming is needed.

**Overflow and unexpected events.** Fail closed. `IN_Q_OVERFLOW` or `Wd < 0` returns
`(true, ids)`, marking every watch mutated (mutation_linux.go:64-67); a read error other than
EINTR/EAGAIN does the same (:57-59); the kqueue drain returns `true` on any kevent error
(mutation_bsd.go:66-68); the unsupported-platform stub returns `true` (mutation_other.go:28).
`observed` latches `c.all` and per-id entries so the answer never regresses (mutation.go:126-140).
None of this is tested: `drain` is a concrete method with no seam, so the overflow claim is
asserted, not pinned.

**kqueue coverage.** `watchedNotes = NOTE_WRITE|NOTE_EXTEND|NOTE_DELETE|NOTE_RENAME|NOTE_REVOKE`
(mutation_bsd.go:24-25). NOTE_WRITE on a directory fires on create, unlink, rename into and rename
out of the watched directory; NOTE_RENAME and NOTE_DELETE cover the directory itself. Coverage is
correct. `EV_CLEAR` plus the cumulative fold in `observed` is the right pairing.

**Residual fail-open, same class, not fixed.** `dirscan.WalkBounded` keeps its own custody window,
from `captureDirectoryState(authority)` before the read (dirscan.go:280) to
`captureDirectoryState(authority)` after the entire subtree has been walked (:317), and rejects on
`sameDirectoryState`. That window spans every child recursion and every callback, and it is guarded
by the change stamp alone. `Read` arms a watch only across its own inner window and drops it on
return. So on a 6.8 kernel a create-then-delete after `Read` returns and before the subtree
finishes is still invisible, in the same walker used by the governance hook's immutable-design
inventory (internal/hook/hook.go:1563), cmd/machinery/sweep.go:108,
cmd/machinery/verify_checkers.go:724 and internal/checker/inventory.go:75. The gates path got this
right (the watch lives in `rootDirectoryWitness` and `revalidateRootDirectory` drains it after the
subtree, confinement.go:446); `dirscan.WalkBounded` did not get the same treatment.

**Watch-descriptor aliasing (minor).** On Linux `inotify_add_watch` returns the same wd for the
same inode on the same instance. If one traversal enumerates the same directory twice on one
channel, the second enumeration inherits the first one's latched mutation. That is a false
positive, so fail-closed, but it is a spurious gate failure with no retry behind it.

---

## 3. Confinement

Clean. Linux arms through `/proc/self/fd/%d` (mutation_linux.go:35-36); the procfs magic link
resolves the descriptor by inode, not by re-walking a name, so an `os.Root` enumeration cannot be
redirected between open and watch. BSD dups the descriptor and never touches a path
(mutation_bsd.go:39). `dir` at the gates site comes from `root.Open(rel)`
(confinement.go:326) and at the oracle site from the already-open `first`. I found no path-based
reopen introduced anywhere in the diff. The only path reopen in the touched code,
`os.Open(inventory.displayRoot)` in `sourceFileInventory.Close`, is pre-existing and unchanged.

One undeclared new dependency: the Linux path requires `/proc` mounted and readable.
A container without `/proc`, or one with `hidepid`/`subset=pid`, now refuses every directory
inventory. That is not in the CHANGELOG's list of refusal causes.

---

## 4. Scope and uniformity

Six entry points arm a channel:

| Site | File:line | Shape |
|---|---|---|
| `readOnce` (dirscan.Read) | internal/dirscan/dirscan.go:107 | one channel **per call** |
| `validateDesignInventoryBounded` | internal/gates/confinement.go:259 | one per traversal |
| `walkTreeDirBounded` | internal/gates/confinement.go:476 | one per traversal |
| `embedResiduePathsBounded` | internal/gates/embedtransaction.go:1313 | one per traversal |
| `walkSourceFilesBounded` | internal/gates/imports.go:136 | one per custody window |
| `readOracleRootDir` | cmd/machinery/oracle.go:796 | one **per call** |

The gates wiring is uniform and good: the channel is a parameter of `readRootDirectory`, the watch
lives in the witness, and revalidation drains the same watch. Three deviations:

- The commit message states "One channel per traversal, not per directory". True for gates, false
  for `dirscan.WalkBounded`, which pays one instance per directory through `Read` and, as above,
  leaves its own outer window unwatched. This is both the churn cost and the residual fail-open.
- Close-error handling is inconsistent: `walkTreeDirBounded` and `walkSourceFilesBounded` join the
  Close error into the return; `validateDesignInventoryBounded` (confinement.go:263) and
  `embedResiduePathsBounded` (embedtransaction.go:1317) discard it with `_ =`.
- Refusal message prefixes differ per site, which is correct and helpful.

**The flagged follow-up is understated.** `internal/install` is described as a file-level
`ChangeID` witness. `walkPluginCacheTopology` (internal/install/install.go:605-648) is a two-pass
**directory** inventory whose per-directory witness is `installFileChangeID(openedInfo)` at :623 and
:643, compared across passes at :578 and revalidated at :581. That is the same directory ABA, the
same coarse clock, in a path that runs during install and plugin-cache retention, and it is
untouched. So within the CHANGELOG's own stated scope ("the fail-closed directory inventories in
`internal/gates`, `internal/dirscan` and the `machinery oracle` inventory ... Every such inventory
now also arms a kernel mutation-event watch"), the claim is inaccurate for `internal/dirscan`
because `WalkBounded`'s outer window is not covered; and a reader will reasonably infer the
directory-ABA class is closed, which install contradicts. That is a misleading claim as written,
not merely an incomplete one.

---

## 5. Tests

**Frozen-stamp reproduction: yes, and it is the right technique.** `changeWitness`
(dirscan.go:196), `inventoryChangeID` (confinement.go:36) and `oracleChangeID` (oracle.go:44) are
seams pinned to a constant, so the three ABA tests reproduce the 6.8 blindness on darwin
deterministically. The claim that they fail on darwin with the mutation check removed is credible
from the code.

**But they are not guarded for Windows.** None of the three test files carries a build constraint.
On Windows the mutation channel is a deliberate no-op (`newMutationChannel` returns `(nil, nil)`,
mutation_windows.go:19; `Watch` returns an unarmed watch, mutation.go:73), so with the stamp pinned
to a constant `TestReadCatchesABAWithABlindChangeStamp` has no witness left and fails. It is latent
today only because ci.yml:56-58 states native Windows test execution is out of scope. The practical
consequence is that "Windows is unchanged and its NTFS ChangeTime is not clock-coarse" is an
assertion with no test behind it, in a change whose entire subject is a witness being blind.

**Refusal coverage: 3 of 6 entry points.** Pinned: `dirscan.Read`
(`TestReadRefusesWithoutMutationWitness`), `readRootDirectory` and `validateDesignInventory`
(`TestReadRootDirectoryRefusesWithoutMutationWitness`,
`TestDesignInventoryRefusesWithoutMutationChannel`), `readOracleRootDir`
(`TestReadOracleRootDirRefusesWithoutMutationWitness`). Not pinned: `walkTreeDirBounded`,
`embedResiduePathsBounded`, `walkSourceFilesBounded` (which also has an untested error path that
must close `rootAuthority`, imports.go:136-139).

**Descriptor/watch release: no test at all.** A repo-wide search for fd counting in tests
(`NumFD`, `/proc/self/fd`, `lsof`, any open-descriptor census) returns nothing. Given that darwin
holds one dup per directory for a whole traversal and Linux holds one watch per directory, and
given that a failed `readRootDirectory` leaves its watch on the channel for the rest of the
traversal, this is the test the change most needs and does not have. `Close` is correctly deferred
on every path in `readOnce` and `readOracleRootDir`, so panics and early returns do release, but
nothing pins it.

**Also untested:** the overflow fail-closed path (no seam on `drain`), and `Mutated` after
`channel.Close()` (returns the last known value rather than an error, mutation.go:129).

`TestMutationWatchReportsRealNamespaceEvents` against the real kernel is a good addition and its
`hasEventChannel` skip is the right shape.

---

## 6. House style and CHANGELOG accuracy

Style is clean. Zero em dashes in the diff, no emoji, no "honest"/"truthful" self-labeling. Comment
prose is dense but explains mechanism rather than restating code.

The Compatibility entry is the weakest text in the change. Two problems:

1. "Unchanged: every supported host ... enumerates exactly as before." Not accurate. Every
   enumeration on Linux and the BSDs now consumes a kernel resource it did not consume before, and
   on the BSDs holds a descriptor per directory for the traversal. That is the compatibility fact a
   reader needs.
2. The list of causes is incomplete. It names an exhausted `fs/inotify/max_user_instances`, a
   kernel without inotify, and an unsupported platform. Missing: exhausted
   `fs.inotify.max_user_watches` (surfaces as ENOSPC, "no space left on device", and is the likelier
   of the two on the gates path); `/proc` unmounted or restricted in a container; RLIMIT_NOFILE
   exhaustion on darwin and the BSDs. The migration line offers only
   `sysctl fs.inotify.max_user_instances=256`.

The Fixed entry's "Every such inventory now also arms a kernel mutation-event watch" is inaccurate
for `dirscan.WalkBounded` as covered in section 4.

---

## Verdict

**HOLD FOR 0.7.1.**

Reasons, in order:

1. The fix is incomplete inside its own declared scope. `dirscan.WalkBounded`'s cross-subtree
   window (dirscan.go:280 to :317) is still stamp-only, and that is the walker the governance hook
   uses for the immutable-design inventory. Shipping a CHANGELOG entry stating the class is closed,
   when the most-used walker in the class still has it, creates an assurance claim the code does not
   support. Closing it means threading one channel through `WalkBounded` and carrying the watch to
   the post-subtree comparison, which is a design change, not a release-day edit.
2. A new deny mode lands on a path that runs on every agent shell and write action, with no retry
   (dirscan.go:69-72), no distinct diagnostic, and no classification separating an environment
   failure from a policy violation (internal/hook/hook.go:886, :946; hooks/machinery-hook.sh:68-71).
   The soak evidence is a single-host container run, not a concurrency run: nothing has exercised N
   agents against one uid's inotify budget.
3. The resource behavior is untested in both directions: no watch or descriptor census, no overflow
   test, refusal pinned at 3 of 6 entry points.
4. The bug being fixed is not a regression. It has been present in every release and requires an
   adversary winning a sub-tick race against the inventory. The remedy, as written, is an
   availability risk on every action. That asymmetry does not favor shipping it on cut day.

If the team elects to ship anyway, the minimum set that removes the brick risk without the
`WalkBounded` refactor is:

- Treat `EMFILE`, `ENFILE` and `ENOSPC` from `newMutationChannel`/`watch` as transient and route
  them through the existing eight-attempt `Read` retry, keeping a hard refusal only for
  `ENOSYS`/unsupported platform.
- Name both sysctls (`fs.inotify.max_user_instances`, `fs.inotify.max_user_watches`) and the
  darwin descriptor limit in the runtime diagnostic, not only in the CHANGELOG.
- Correct the Compatibility entry: drop "enumerates exactly as before", add ENOSPC/max_user_watches,
  `/proc` unavailability, and RLIMIT_NOFILE on darwin and the BSDs.
- Guard the three frozen-stamp tests off Windows, or assert the Windows behavior explicitly.
- Add refusal tests for `walkTreeDirBounded`, `embedResiduePathsBounded` and
  `walkSourceFilesBounded`, and one release test that watch and descriptor counts return to
  baseline after a traversal.
- Restate the follow-up: `internal/install`'s `walkPluginCacheTopology` is a directory inventory
  with the identical blindness, not a file-level witness.
- Either fix `dirscan.WalkBounded`'s outer window or say plainly in the CHANGELOG that it is not
  covered.

**Single most important risk:** the change ships a claim that the directory ABA is closed while
`dirscan.WalkBounded`, the walker behind the governance hook's own design inventory, still proves
its cross-subtree window with the same blind stamp on the same 6.8 kernels the entry was written
for. Everything else on this list is correctable in an hour; that one is a design gap that will be
believed fixed.

### 2026-09-09T16:55:43Z ramirosalas
2026-09-09 evidence: make ci-linux (pinned golang:1.27.1-trixie image, 2 CPUs, linux/amd64, Ubuntu 24.04 host kernel 6.8, Docker 29.7.2, run as deploy) fails TestInventoryRejectsContinuousDirectoryGrowth and TestInventoryRejectsSameDirectoryABADespiteRestoredMtime in scripts/tree-inventory at 0.00s, exactly the container directory-witness blindness this item describes, while the hosted ubuntu-latest test job on the same commit (98b47e3, 0.7.1 candidate) passes them. So the containerized sweep is not yet a faithful mirror of the hosted test job for this package; the fix on fix/aba-linux (af54c9f, HOLD after review) or a fine-grained-timestamp tmpfs for the test temp dirs in scripts/ci-linux.sh is needed before ci-linux can claim hosted-equivalent evidence.
