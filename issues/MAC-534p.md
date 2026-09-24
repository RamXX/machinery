---
id: MAC-534p
title: "Possibility properties: refute AG EF claims and trap SCCs structurally on the guard-erased machine graph"
status: open
priority: 1
type: feature
labels: [formal, g3, lint, liveness]
created_at: 2026-09-24T21:27:28Z
created_by: ramirosalas
updated_at: 2026-09-24T21:27:28Z
content_hash: "sha256:84b57c222d5190ec7b02af4fd3fc6f3fc79a281a18c3e2a53511403522c2419c"
---

## Description
Machines have no check for possibility properties (CTL AG EF P: from every reachable state, some path still reaches P). Lint checks local dead ends (internal/lint/lint.go:359, a non-final leaf needs a transition) and forward reachability from the initial state (lint.go:835). TLC checks Overlay ~> Domain (internal/tla/tla.go:549) and deadlock freedom. None of them catch a trap: a strongly connected set of non-final states with no path to any final state passes lint when each member has a transition. Claims like "an order can always reach a terminal state" or "a user can always cancel" are unchecked.

Source: Hillel Wayne on TLA+ limits (2026-09): reachability/possibility properties are not expressible as linear-time formulas over single behaviors; they need a branching-time (CTL) check.

Soundness constraint: the generated TLA+ erases guards to nondeterminism, which over-approximates behaviors. On the guard-erased graph a path's existence does NOT prove possibility (a guard may block it), but its absence DOES refute it. So the gate is refutation-only: deterministic FAIL when the claim is structurally impossible; a positive result stays an attested claim (Gv), never "proven".

Acceptance criteria:
1. Default structural check in `machinery check` (G3 lint family): every state reachable from the initial state can reach some final state on the guard-erased graph, unless the machine is a declared perpetual envelope (the existing no-final case in tla.go:538-546). A trap SCC fails naming its member states.
2. A declared possibility claim form on a machine (e.g. from every state, event E is enabled on some path, or state S is reachable) is refuted deterministically when no guard-erased path exists, with a counterexample state.
3. A passing possibility claim is reported as structurally possible, not proven; docs and the generated spec header say so.
4. Unit tests for trap SCC, nested/compound states, parallel regions, perpetual envelope exemption; one example design fixture exercising a declared claim.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
