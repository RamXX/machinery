# M3 - Optimizer slice

## Outcome

The pure optimizer deterministically selects the feasible portfolio with the MINIMUM exact
historical maximum drawdown: exactly 16 Holding records over 16 distinct canonical candidate
identities, integer nonnegative weights in basis points summing to exactly 10000 (zero-weight
records retained, never dropped or replaced), and the stored drawdown. It compares exact
rational drawdowns BEFORE any rounding, stores `maxDrawdown = floor(10000 x D + 1/2)` basis
points (nearest, ties upward; valid 0..10000 inclusive; intermediates never rounded), and
breaks exact-score ties by the ascending ASCII ticker vector, then the weight vector in that
ticker order.

## Domain context

Inputs are one deduplicated CandidateSet and complete bounded price histories. The value rule is
normalized buy-and-hold: for observation t, `V[t] = sum over i of (weight_i/10000) x (P[i,t]/P[i,0])`,
quantities fixed across the window, so V[0]=1 and there is no periodic rebalancing;
`peak[t] = max(V[0..t])` and `D = max_t (peak[t]-V[t])/peak[t]`, including t=0. Prices are
strictly positive finite exact decimal strings matching `(0|[1-9][0-9]*)(\.[0-9]+)?` on the
unadjusted-close basis (`100` and `100.00` are equal prices); the matrix declares exactly
`lookbackDays` strictly increasing distinct real Gregorian dates (YYYY-MM-DD, years 0001-9999)
common to every candidate, with positional alignment against that single declared vector.
Admission is whole-input: reject the ENTIRE universe when any candidate history is invalid -
a missing row or explicit null cell is INCOMPLETE_HISTORY; a present row of the wrong length,
malformed value, extra row, mixed price basis, or noncanonical/duplicate identity is
INVALID_INPUT; fewer than 16 candidates is INSUFFICIENT_CANDIDATES; a violated
OptimizerLimits{maxCandidates>=16, maxLookbackDays>=2, maxScalarBytes} bound is
LIMIT_EXCEEDED, checked before decoding; all return InfeasibleError with the closed reason
set, validated in that order. Outputs are Portfolio holdings and drawdown. Enforce
`portfolio-size-16`, `portfolio-holdings-deduped`, `portfolio-from-candidates`,
`portfolio-has-drawdown`, `holding-weight-nonneg`, and `holding-weights-sum-full`.

## Architecture context

Own only pure code in `pf.optimizer` plus domain value construction. No filesystem, network,
environment, random global state, or wall clock is allowed. The RecommendationRun adapter invokes
this API but its lifecycle is outside this packet.

## Behavior and oracles

This slice is pure data refinement rather than a separate lifecycle machine. Treat the six named
invariants as the executable oracle. For equal exact objective values, compare the whole
ascending ticker vector first, then the whole weight vector in that ticker order, and choose
the least key; IDs, insertion order, names, sectors and time never break a tie. Reject
missing, malformed, misaligned or insufficient histories per the whole-input admission rule;
"fewer than 16 full histories" is one sufficient failure example, not the whole rule.

## TDD and implementation

Write generated and hand-selected property cases before implementation: permutations produce the
same result, duplicates cannot survive, weights remain bounded and sum exactly under the chosen
numeric representation, and malformed input fails closed. Lock RED, then implement the transform
with stable sorting and no ambient inputs.

## Risks and recovery

Avoid floating-point equality as an acceptance criterion; the contract requires exact rational
comparison with no tolerance, and a backend unable to preserve the ordering must fail
explicitly, never guess a tie. Candidate count and history length are bounded by the explicit
caller-owned OptimizerLimits, not a hidden tolerance or ambient capacity. The function has no
side effects and no repository/operation-handle access - it never queries storage or clock -
so recovery is retry with the same immutable input.

## Acceptance

Run the six invariant properties repeatedly with fixed seeds plus malformed-input and permutation
tests. Record command output, seed inventory, and implementation check in `acceptance/M3.yaml` for
the reviewed commit.
