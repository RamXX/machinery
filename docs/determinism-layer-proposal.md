# Proposal: the determinism relational layer (gate Gd)

**Status: proposal, not implemented.** Nothing in the CLI reads this yet.
Recorded now because the shape was derived from a real design and should not
be re-derived later from memory.

## The defect class

Three separate findings in one design turned out to be one property:

- A fetch that fails could be configured to mean "clean".
- A semantic option could arrive as a configuration flag and never be
  recognized as an input to a result.
- A withdrawn override made its findings vanish, and nothing in the record
  distinguished that from an improvement.

The common shape: **an input that can change a result was not recorded with
the result.** That is structural, it lives in the domain model, and it is
therefore checkable, which is the same argument that produced the G2
acyclicity check in v0.3.6.

The existing layers cover neighbouring properties. Policy governs who may
act on a record. Integrity governs what structures are admissible.
Isolation governs what a record may reference across a tenant boundary.
None of them asks whether a result can be reproduced and explained.

## Generic vocabulary

Nothing below is domain-specific. Any design that produces results a reader
must be able to trust has these parts: a price, a routing decision, an
authorization, a risk score, a diagnosis, a compliance verdict.

- **Outcome**: an entity whose value must be reproducible and explainable.
- **Disclosure**: where an outcome's provenance is recorded. Either `self`
  (the outcome carries its own binding) or a named entity (a run record, a
  manifest, a scope statement).
- **Influence**: an entity whose state can change an outcome.
- **Binding strength** (`via`): how firmly an influence is tied to the
  outcome. Four kinds, and they are not interchangeable:

  | via | Guarantee | Precondition on the influence |
  |---|---|---|
  | `hash` | Content-addressed; the exact input is provable and replayable | outcome carries a hash attribute binding it |
  | `reference` | The input row is identifiable later | influence is immutable or versioned, else the reference pins nothing |
  | `as_of` | Reconstructible by time | **influence must be bitemporal** |
  | `disclosure` | Stated in the rendered result, not machine-resolvable | none; the weakest |

## The property

For every declared influence `I` of outcome `O`, `I` is reachable from
`O`'s disclosure at no less than the declared binding strength, **checked
over the transitive closure** rather than direct edges. Influence composes:
if A can change B and B can change C, then A can change C, and real drift
arrives transitively (new content reaching an outcome through a release or
an upgrade, not through any edge anyone declared). A direct-edge check
would pass the defect that motivates the layer.

## Two things that differ from the isolation layer

**Disclosure is per-outcome, not global.** Isolation assumes one tenant
entity. Determinism cannot: in the worked example one outcome binds its own
input hashes and replays byte-identical, while another discloses through a
shared run record. Declare disclosure per outcome.

**Reconciliation runs in both directions.** Isolation reports residuals one
way: present in the model, not declared. Determinism needs both.

- *In the model, not declared*: a new entity that can reach an outcome and
  nobody classified it. Drift.
- *Declared, not in the model*: an influence with no entity behind it,
  which is precisely a semantic option living as configuration.

The second direction is the one that earns the layer. **Declaring an
influence obligates it to be domain data**, with the versioning,
attribution, and disclosure that implies. Without it the layer only checks
the influences that already behaved.

Both directions need the waiver idiom the policy layer already has
(`residuals waived with reason`), because transitive reach produces a long
tail and an unwaivable check becomes noise.

## The precondition check is where the real value sits

The layer should verify not just that a binding is *declared* but that the
declared strength is *achievable given the influence's own shape*. The
`as_of` row is the sharp one: **declaring `via: as_of` on an influence that
carries no validity time is a defect the model can prove**, because the
date cannot reconstruct what the influence was.

That is not hypothetical. In the design this proposal was derived from,
exactly that defect existed and had survived every prior review: an outcome
was documented as a function of three inputs, bound a date, and one of
those inputs was not bitemporal, so two outcomes with the same date and
different inputs differed with nothing in the record to say so. The layer
would have failed it mechanically on day one.

## Build plan

Staged, cheapest value first. Phases 1 and 2 deliver most of the benefit
with no solver work at all.

1. **Declaration format, bidirectional reconciliation, gate.** Parse the
   layer file, reconcile against the model, report residuals both ways with
   waivers. No Alloy. Catches drift and the configuration escape
   immediately.
2. **Binding-strength preconditions.** Pure model analysis: `as_of`
   requires a bitemporal influence, `hash` requires the outcome to carry a
   binding attribute, `reference` requires immutability or versioning.
3. **Alloy encoding.** Transitive closure over the influence graph,
   counterexamples for unreachable influences, `machinery alloy` emits it
   alongside the existing three.
4. **Oracle generation.** One row per (outcome, influence) with the
   required binding strength, consumed by conformance tests exactly as the
   tenant oracle is.
5. **Docs and a bundled example**, plus adversarial entries in the
   experiments registry, per the standing rule that review findings convert
   1:1 into entries.

## Open questions to settle before phase 1

- **Candidate detection.** Isolation reconciles declared references against
  the model. For influence, "what could reach this outcome" is a broader
  question, and inferring candidates from relationship structure alone may
  produce more noise than signal. Decide whether residual detection is
  structural (everything reaching an outcome transitively) or declaration
  anchored, and expect the waiver idiom to carry the difference.
- **Naming.** `determinism` describes what it protects; `influence`
  describes what it declares. Gate letter `gd` is free.
- **Second falsification.** The shape came from one design. Author a
  declaration for a bundled example before the format hardens, because a
  format that fits only the design that invented it is not a format.
