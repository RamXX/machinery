---
id: MAC-2f7g
title: "Invariant property class (safety, liveness, possibility, hyper, statistical, informal) and Gc carrier compatibility"
status: open
priority: 2
type: feature
labels: [formal, gc, invariants]
created_at: 2026-09-24T21:27:28Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:d26c9d3ef0e4ec2a7b3e3bbf4346366eee7458af2b831e379efd3d7ad1621db5"
blocked_by: [MAC-534p]
related: [MAC-0b1e]
---

## Description
Most properties people care about cannot be written as a formula a model checker evaluates over single behaviors. Machinery states this in prose ("The methodology names each residual in the design artifacts", README "What machinery does not verify") but no gate enforces that a green check never implies coverage of a property the carrying engine cannot express.

Source: Hillel Wayne on TLA+ limits (2026-09): TLA+ covers safety ([]P) and liveness (<>P, []<>P, P ~> Q) over single behaviors; it cannot express possibility (needs CTL), hyperproperties (two or more traces: noninterference, "users cannot infer secrets"), statistical properties (p95 latency, needs PRISM-style or load testing), or properties with no logical formula at all.

Proposal: each invariant carries a property class: safety | liveness | possibility | hyper | statistical | informal. Gc-carrier (every invariant has a named carrier or a reasoned waiver) additionally checks carrier/class compatibility:
- TLC carries safety and liveness; Alloy carries static relational safety;
- possibility routes to the structural refutation check (see linked story) plus a Gv attestation for the positive side;
- hyper routes to a named test obligation or an external checker (Gk) that declares hyperproperty support, never TLC/Alloy;
- statistical routes to a named load/benchmark obligation;
- informal routes to a Gv attestation.
An incompatible pairing (e.g. a hyper invariant carried only by a TLC proof) fails Gc.

Open design question: Modelith owns the invariant schema. Decide whether the class lives in Modelith (upstream change) or in a machinery-side annotation keyed by invariant id; the annotation route mirrors the policy/integrity/isolation layers.

Acceptance criteria:
1. A decision recorded on where the class lives, with the alternative considered.
2. Class vocabulary closed; unknown class fails loudly.
3. Gc rejects each incompatible carrier/class pair with a message naming the invariant, the class, and the carriers that could hold it.
4. Absent class: default behavior decided and documented (warn during a migration window, then required), recorded in CHANGELOG compatibility notes.
5. Tests per class/carrier pair; one example design migrated.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Decide where the class lives (Modelith vs machinery-side annotation keyed by invariant id), closed vocabulary, Gc compatibility matrix, default/migration rule, tests, one example migration. Evidence: No property-class vocabulary exists in code or docs (grep for 'property class' and hyperproperty finds only nd issue files). Gc carrier check exists but has no class compatibility. Notes: blockedBy MAC-534p is only partly real: the possibility->structural-refutation routing needs 534p, but the class vocabulary and the other class routings do not. Could be split so the rest ships first.

## History
- 2026-09-24T21:33:56Z dep_added: blocked_by MAC-534p

## Links
- Blocked by: [[MAC-534p]]
- Related: [[MAC-0b1e]]

## Comments
