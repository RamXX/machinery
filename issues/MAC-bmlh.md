---
id: MAC-bmlh
title: "Hook design-snapshot check deadlocks the seat when a process writes into the governed tree"
status: open
priority: 1
type: bug
labels: [hook, governance, h2, from-next]
created_at: 2026-09-24T21:32:49Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:49Z
content_hash: "sha256:968f55a69b2300145ad7a3f7b59a4c22cf859254b98ffb3ad7dcbba816b53e80"
---

## Description
Hook design-snapshot check deadlocks the seat when a process writes into the governed tree

Problem (H2, 2026-09-19 to 2026-09-20): the design-inventory fingerprint races any process writing into the tree. H2's shipped wall script logs beside itself, in-tree by default. While a wall ran, the hook's snapshot check saw a growing log, failed closed, and denied EVERY later tool call of the agent, including the kill commands that would have stopped the writer. Only a human outside the seat could break the deadlock. Workaround: run the script from a copy outside the tree.

Evidence: internal/hook/hook.go:804-832 (`withRoutingSnapshot`) acquires `gates.AcquireSnapshot(designDir)` around every routed event and returns `snapshot.CheckUnchanged()`, so any concurrent byte change under the design tree fails the event closed. Nothing in CHANGELOG 0.9.0 to 0.10.1 addresses concurrent writers (0.9.0 only stopped fingerprinting the tree for new pending tokens).

Proposed fix (one or more):
- Scope the fingerprint to the design inventory it actually guards (design sources, not arbitrary files under the tree), honoring `.machineryignore`.
- Tolerate declared transient paths (logs, caches) listed in a config or `.machineryignore`.
- Never deny the process-control verbs an agent needs to stop a writer (kill, pkill, stopping a background job) on a snapshot-changed failure; retry-with-backoff on a changed snapshot before denying.
- Document that tools writing inside the governed tree must log outside it.

Acceptance criteria:
1. A reproduction where a background process appends to a file under the design tree while the agent issues tool calls: calls touching unrelated paths are not denied, or are denied with a message naming the changing path.
2. A kill or stop command issued while the snapshot is changing is admitted.
3. A path listed as transient (or ignored) never fails the snapshot check; a change to a real design source during an event still fails closed.
4. Existing hook snapshot tests stay green and the fail-closed direction for design sources is preserved.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
