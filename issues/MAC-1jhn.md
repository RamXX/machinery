---
id: MAC-1jhn
title: "assuranceflow: a reader of ledger/heads sees a publisher's in-flight .publish-<nonce> entry and fails INVALID_SCHEMA"
status: closed
priority: 2
type: bug
labels: [flaky-under-load, assuranceflow, tdd-store, concurrency]
created_at: 2026-09-10T07:34:53Z
created_by: ramirosalas
updated_at: 2026-09-24T21:31:55Z
content_hash: "sha256:9b7b3b98bd0ca2078504206c4b4d531f87a3becb65cdc33c3a879c0d1e73a248"
closed_at: 2026-09-24T21:31:55Z
close_reason: "Fixed by e2db8e49 in 0.8.0 (reserved .publish-<nonce> predicate in internal/tdd/store.go, option 1 of the AC). Triage 2026-09-24."
---

## Description
## Symptom

`TestRegisterConflictingWriters` fails with a schema rejection, not a contention outcome:

```
--- FAIL: TestRegisterConflictingWriters (0.06s)
    register_test.go:558: unexpected error: INVALID_SCHEMA: ledger/heads/ holds unknown entry ".publish-ae571e091ad24322"
FAIL	github.com/RamXX/machinery/internal/assuranceflow	0.834s
```

Observed 2026-09-10 on a native linux/amd64 host (kernel 7.0.0-30, 4 CPUs) running the full
non-race suite in a container via `dagger call golden-nightly`. An immediate re-run of the same
tree passed 80/80, and the test passes 25/25 in isolation on darwin/arm64, so it is load and
interleaving dependent.

## Root cause

The publisher stages its temp file inside the very directory it publishes into:

- `internal/tdd/store.go:173` -- `tmp := filepath.Join(dir, fmt.Sprintf(".publish-%s", hex.EncodeToString(nonce)))`
  where `dir := filepath.Dir(finalPath)`, followed by `durableWriteBytes`, `os.Chmod`,
  `os.Link(tmp, finalPath)` and a deferred `os.Remove(tmp)`.
- `internal/tdd/store.go:592` -- the `ledger/heads/` enumeration rejects any entry it does not
  recognize as a head: `INVALID_SCHEMA: ledger/heads/ holds unknown entry %q`.

So for the window between `durableWriteBytes` and the deferred `os.Remove`, a concurrent reader
enumerating `ledger/heads/` sees the writer's private in-flight entry and fails closed on it.
`TestRegisterConflictingWriters` runs concurrent writers by construction, which is why it is the
test that catches it.

## Why it matters

This is the fail-closed direction, so nothing is silently corrupted, but it is still a real
defect rather than test noise: a reader of a healthy ledger reports a schema violation purely
because another writer was mid-publish. Any concurrent consumer of `ledger/heads/` can hit it in
production use, not only the test.

## Precedent in this repository

The same class is already solved on the design side. `internal/designlock` has
`designlock.AcquireReader`, which exists precisely because "a raw Walk can otherwise capture the
private publication sentinel or a cross-file partial state" (see the comment on `copyDesignTree`
in `internal/hook/hook_test.go`). The ledger reader has no equivalent: it enumerates raw.

## Options (design decision, not yet made)

1. Reader-side: have the `ledger/heads/` enumeration skip in-flight publication entries by their
   reserved prefix, and keep rejecting genuinely unknown names. Cheapest, but it widens what the
   reader tolerates and needs the prefix to be a reserved, documented namespace.
2. Writer-side: stage outside the enumerated namespace (a sibling `staging/` directory on the
   same filesystem) and link into `heads/` atomically, so no reader can ever observe a partial
   state. Keeps the reader strict, which matches how the rest of the codebase treats closed
   namespaces.
3. Reader-side coherent snapshot: give the ledger the equivalent of
   `designlock.AcquireReader`, so a reader takes a governed snapshot rather than a raw walk.
   Most consistent with the existing design, most work.

Option 2 or 3 preserves the closed-namespace property that the `INVALID_SCHEMA` check exists to
enforce; option 1 weakens it.

## Relationship to MAC-y5ho

Same test, different race. MAC-y5ho is `mkdir .../staging/writer.lock` returning EINVAL under
load. This one is a reader observing a publisher's in-flight entry. Fixing either does not fix
the other, and MAC-y5ho's title understates the scope of what that test is catching.

## Reproduction

Run the full non-race suite on a loaded multi-core Linux host, ideally in a container:

```
dagger call golden-nightly          # or: go test -count=1 ./... -timeout=20m under load
```

It is intermittent. A targeted loop under concurrent load is more likely to hit it than an
isolated `-count=N`, because the window needs another writer publishing while a reader enumerates.

## Acceptance criteria

- A concurrent reader of `ledger/heads/` never observes a publisher's in-flight entry, proven by
  a test that interleaves a publish with an enumeration deterministically (a test hook at the
  point between `durableWriteBytes` and `os.Link`, in the style of
  `testAfterFingerprintFileRead`), not by a load-dependent sweep.
- The `INVALID_SCHEMA` rejection still fires for a genuinely unknown entry in `ledger/heads/`,
  with a regression test.
- `TestRegisterConflictingWriters` passes under a loaded containerized run of the full suite.
- If option 1 is chosen, the reserved prefix is documented as a closed namespace and the
  reader rejects every other dot-entry.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:31:55Z status: open -> closed

## Links


## Comments
