# Formal verification of the go-crm machines

Every model here is checked exhaustively by TLC (`make verify-formal`). No claim is
by inspection. Three layers, matching the correctness ladder.

## Control-flow models (generated)

`Deal.tla`, `Task.tla`, `User.tla`, `Session.tla`, `CommandExecution.tla` are generated
from the machine JSON by `tla_gen.py`. The state graph is exact; guards are
over-approximated (both branches of a guarded event are allowed), which is sound for
the properties below because they hold over a superset of the real behavior. The
standard persist-retry overlay is modeled with a bounded counter so liveness is decidable.

Each proves:
- `TypeOK` (safety): states are valid and the retry counter never exceeds its bound.
- `Live_OverlayResolves` (liveness): from any transient overlay state the machine always
  eventually returns to a resting domain state. No stuck half-persist, no infinite retry.
- deadlock-freedom (TLC default; `final` states stutter via `Terminated`).

State counts: Deal 18, Task 16, User 14, Session 24, CommandExecution 31.

## Data-refined domain invariants (hand-annotated)

`DealData.tla` adds the committed stage and the persist context that the control-flow
model abstracts away, so the real domain contract is checked, not just reachability:
- `Inv_StageValid`, `Inv_DomainConsistent` (well-formedness),
- `Inv_Atomic` (persist atomicity: the committed stage flips only on the atomic commit; a
  failure rolls back unchanged),
- `Inv_WonHasCloseDate` (the `deal-won-has-closedate` invariant),
- `StageForward` (the `deal-stage-forward` invariant: forward, to Won/Lost, or reopen only),
- `Live_OverlayResolves` (every persist attempt terminates).

Checked over 200 states. This is the hand-annotated refinement on top of the generated
skeleton; the action semantics live in the code, not the machine config, so this layer
is authored, not generated.

## Refinement (the recursion substrate)

`DealContract.tla` is the ABSTRACT contract the big picture assumes of a deal aggregate:
resting-or-busy, atomic while busy, and every busy period terminates. `DealRefinement.tla`
defines a refinement mapping from the concrete `DealData` state to the contract and checks
with TLC that every concrete behavior is a contract behavior (`RefSpec`) and that the
abstract liveness holds (`RefTermination`).

This is the composition rule for large systems: a subsystem is designed fresh at its own
level and is a correct part of the whole exactly when it refines the contract the big
picture assumed of it. Verifying each part against a small contract, rather than the
flattened system, is what makes the approach scale.

## Run it

```sh
make verify-formal        # from the repo root: generate + check everything
```
Requires Java 11+. `tlc.sh` fetches `tla2tools.jar` into `~/.cache/machinery` on first use.
