# M3 - Optimizer slice

## Outcome

The pure optimizer deterministically selects exactly 16 unique candidates with non-negative weights
summing to the full allocation and reports the portfolio drawdown.

## Domain context

Inputs are one deduplicated CandidateSet and complete bounded price histories. Outputs are Portfolio
holdings and drawdown. Enforce `portfolio-size-16`, `portfolio-holdings-deduped`,
`portfolio-from-candidates`, `portfolio-has-drawdown`, `holding-weight-nonneg`, and
`holding-weights-sum-full`.

## Architecture context

Own only pure code in `pf.optimizer` plus domain value construction. No filesystem, network,
environment, random global state, or wall clock is allowed. The RecommendationRun adapter invokes
this API but its lifecycle is outside this packet.

## Behavior and oracles

This slice is pure data refinement rather than a separate lifecycle machine. Treat the six named
invariants as the executable oracle. For equal objective values, use the declared ticker ordering
as the total deterministic tie-break; reject missing, non-finite, or insufficient histories.

## TDD and implementation

Write generated and hand-selected property cases before implementation: permutations produce the
same result, duplicates cannot survive, weights remain bounded and sum exactly under the chosen
numeric representation, and malformed input fails closed. Lock RED, then implement the transform
with stable sorting and no ambient inputs.

## Risks and recovery

Avoid floating-point equality as an acceptance criterion; use the domain's declared tolerance or
exact fixed representation. Bound candidate count and history length. The function has no side
effects, so recovery is retry with the same immutable input.

## Acceptance

Run the six invariant properties repeatedly with fixed seeds plus malformed-input and permutation
tests. Record command output, seed inventory, and implementation check in `acceptance/M3.yaml` for
the reviewed commit.
