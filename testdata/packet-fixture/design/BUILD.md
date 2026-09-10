# Deal desk build

Mode: manifest
Linkage: matrix

## 1. Purpose and scope

A small manifest-mode design that exists to exercise `machinery packet`: one shard, two
milestones, a traceability matrix, and a protocol section a slice can cite from the root.

## 6. Traceability matrix

| invariant id | enforcement | owner | test obligation |
|---|---|---|---|
| `deal-stage-forward` | `guardCanAdvance` on `Deal` | crm.domain | property test over every stage pair |
| `rbac-write-scope` | `guardCanWrite` on `Deal` | crm.domain | negative test per role |

## 9. Build plan

**M0 - Walking skeleton.** A deal advances one stage end to end.
Demo: a deal advances from Lead to Qualified.
Shard: [core](BUILD/core.md)
DoD: DEAL-eb0c40 green against the real store.
NFR: security (rbac-write-scope), capacity (one writer), observability (advance counter).

**M1 - Deal lifecycle.** Win and lose, with authorization held to the policy oracle.
Demo: a deal is won, a deal is lost, and a clerk is refused.
Shard: [core](BUILD/core.md)
DoD: DEAL-38ba11 and DEAL-1fe825 green, ORACLESET{formal/Policy.oracle.md} green.
Slice narrative, written in prose and binding nothing: slice M1-S1 takes the win path and
slice M1-S2 takes the lose path together with the policy rows. This paragraph is what the
projector must never read.

## 11. Hard-TDD protocol

1. Lock the RED test on the stable id before writing any code.
2. One transition row is one test case; key it on the stable id.
