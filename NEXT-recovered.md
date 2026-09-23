# NEXT (recovered): undelivered entries from the pre-0.9.0 NEXT.md

Recovery note (2026-09-22): the 789-line NEXT.md read earlier today was an untracked scratch file.
The lane that shipped 0.9.0 replaced it with a tracked 9-line file, so every entry below vanished
from disk and from history. This file restores the entries 0.9.0 did NOT deliver, verbatim from the
earlier read. Delivered in 0.9.0 and therefore omitted: 28 (host-denied pending token expiry),
29 (Gr-reads), 30 (event payload twins), 31 (lane-scope protocol in docs/packet-projection.md),
32 (fixture obligations in packets), 34 (Gz-authz, shipped inside Gx), 35 (named-unit fact
resolution), 36 (VALUES closed vocabularies). 37 is delivered only for payload cells; its residual is
retained below. Entries 1-3 are recorded premises, not work.

Owner policy (2026-09-03): group changes, weekly cadence, no daily releases. Untracked scratch list;
process into issues, commits, and release notes as a batch. Sources: the nil trust-chain design session of
2026-09-03 and the H2 identity revision of 2026-09-04. Reviewed against machinery v0.6.11 on 2026-09-05;
delivered entries are removed rather than retained as release history.

## 1. --skip-plugins keeps fail-closed plugin ownership discovery

Resolved policy, retained as the recorded premise: `--skip-plugins` skips only host plugin management;
plugin-ownership discovery still runs before the skip is honored, because discovery decides which homes
the plugin serves, and its errors abort rather than degrade to a warning. There is no unsafe fallback to
defaults; docs/agent-portability.md now states this contract, including that receipt and ownership
inspection still run under `--skip-plugins`. No further change owed unless the policy itself is reopened.

## 2. Registry inputs must be registry-relative; repo-root --registry is the supported workaround

Registry `inputs` resolve against the directory containing the registry file, with no `..` allowed, so
the default `.machinery/checkers.local.yaml` forces a copy of the committed adapter under `.machinery/`.
The supported workaround is documented in docs/external-checkers.md: place the registry beside the
committed inputs (e.g. repo root) and pass `--registry`, which removes the copy without changing the
derived closure. Allowing repo-root-relative sources from the default location remains an unapproved
feature extension (same closure either way); not authorized in this batch.

## 3. runtime_closure pins one platform; multi-arch CI cannot reproduce natively

`checker.runtime_closure` pins ONE platform, so a design verified on linux/arm64 cannot be reproduced on
an amd64 CI runner without a manifest re-pin. Documented accurately in docs/external-checkers.md: the
declared `--platform` is always used, and a daemon with emulation (Rosetta, qemu/binfmt) reproduces the
pinned userspace, emulated reproduction, explicitly not native-host test evidence. A per-platform
closure list (one digest per platform, each bound) remains an unapproved feature extension.

## 4. External-checker evidence needs four first-class outcomes with a binary hook boundary

`evidence_schema` 1.0 collapses checker results to `pass` / `fail`: an undecided element must be omitted
and therefore becomes indistinguishable from a checker coverage gap, while `not_applicable` can only be
approximated by a static manifest residual. That loses material semantics. An unknown result means the
checker evaluated the element but lacks a deciding fact or judgment; a coverage gap means it never
decided the element. Applicability can also vary by subject, facts, and evaluation time, so it is not a
waiver and cannot truthfully live in the manifest.

Add an engine-neutral four-state evidence contract, without naming or depending on any particular
checker implementation:

- introduce a new evidence schema version whose top-level and per-element outcomes are `pass`, `fail`,
  `unknown`, and `not_applicable`; split the evidence and projection schema-version constants so the
  unchanged projection contract can remain 1.0;
- keep reading evidence 1.0 during migration, but require committed and freshly reproduced evidence to
  use the same schema and preserve the complete outcome rows byte-for-byte;
- count every explicit outcome as coverage; only an absent claimed element is a coverage gap. Keep
  manifest residuals solely for declared, reasoned exclusions from the checker's obligation set;
- validate aggregation deterministically: any `fail` dominates, otherwise any `unknown` yields
  `unknown`, otherwise any applicable `pass` yields `pass`, and an all-`not_applicable` set yields
  `not_applicable`. A pass may not hide a failed or unknown row, and `unknown` must surface as REVIEW
  REQUIRED rather than compliance;
- preserve the four-state result in `machinery check`, evidence, findings, replay, and
  `verify-checkers`. At a host hook or other binary policy boundary, collapse fail-closed: `pass` and
  `not_applicable` permit the action, while `fail` and `unknown` deny/block it; the detailed reason
  remains available to the agent and audit trail;
- keep Machinery checker-agnostic: schemas, Go types, CLI text, docs, tests, and shipped fixtures use
  only generic outcome vocabulary and a synthetic checker. Any vendor, rule-language, corpus, or engine
  identity remains confined to the private adapter and git-ignored local registry; committed generic
  evidence carries hashes and opaque attestations, not implementation names.

Acceptance tests must distinguish all four outcomes, distinguish `unknown` from a missing coverage row,
prove fact/time-dependent `not_applicable` without a manifest edit, reject inconsistent aggregate/row
combinations, reproduce all outcome rows exactly, retain 1.0 compatibility, and prove the
Claude/Codex/OpenCode adapters emit only their existing allow/deny or block/no-output protocols with
fail-closed binary reduction. Unapproved feature scope.

## 5. Policy algebra with set-valued subject holdings (H2 ask)

H2 models composable `TenantRole`s (a principal's held capability set is the union of its assigned roles),
but the relational policy layer only admits one role attribute per subject (`role_attr: preset`), so the
capability re-base is staged "behind a machinery enhancement" (H2 DECISIONS 2026-08-31). Extend
`policy.relational.yaml` to a set-valued holding (subject holds a set of register keys; grants are keyed
by capability, not by role), regenerate the oracle over capability rows, and keep the preset form as a
degenerate case so existing designs compile unchanged. Unapproved feature scope.

## 6. Per-consumer READS declaration on event fan-outs (H2 ask): residual generalization

The per-edge `reads` override was delivered and accepted; the residual is the H2-side adoption: three
fan-outs in H2 still declare at the intersection with a prose note. Migrate those declarations to
per-consumer-row overrides so the declared read set is exact per edge and the G2/Gx carried-field checks
bind it. (Delivered capability; remaining work is consumer-side, listed here so it is not lost.)

## 7. Hierarchical isolation scoping (H2 headroom): intentionally deferred

`isolation.relational.yaml` admits exactly one tenant entity with `lone` subject membership on the
subject's own attribute. H2's 2026-09-04 identity revision kept `Principal` as the subject (Identity is
tenantless and never acts), so it needed no change here; the open case is `BusinessUnit`, a lens today
that may become an access boundary post-v1. Extend the annotation to a scoping hierarchy
(tenant > unit) with references checked at the declared level, keeping today's single-level form as the
degenerate case so the generated Isolation.als and oracle stay byte-identical for existing designs.
Explicitly deferred until a concrete access-boundary need exists; do not pull forward.

## 8. Tenant-scoped uniqueness in the integrity layer (H2 ask)

`integrity.relational.yaml` unique rows are (entity, attribute, invariant) with no scope, so a
per-tenant uniqueness invariant such as H2's re-scoped `principal-email-unique` (unique within one
Tenant, the same Identity holding one principal per tenant) cannot be stated and had to leave the layer
(H2 DECISIONS 2026-09-04), carried by a composite index plus a matrix contract instead. Add an optional
`scope:` (an entity the uniqueness is keyed under, resolved through the record's tenancy attribution) so
scoped uniqueness compiles to Alloy and the oracle, with the unscoped row as the degenerate case.
Unapproved feature scope.

## H2 delivery lessons: proposed workstreams (2026-09-14)

Owner requested concrete future Machinery work here after H2 exposed erasure,
recovery and creation-contract gaps despite repeated design reviews. These are
proposals to refine into Machinery-owned issues, not implemented capabilities,
new H2 gates, an upgrade request, or proof of an engine defect. Preserve the
weekly release policy and reconcile with the live backlog before creating issues.
Priority is 9, then 10/11, then 12; each can ship independently.

## 9. Make design readiness and its evidence limits explicit

Problem: structural consistency, control-flow model checking, semantic review,
static test attribution and successful runtime verification can all look like
"green" while proving different things. H2's restored-database key recovery
problem survived a design that consistently named key destruction. Reviewers
need to see which external assumptions remain untested before calling a design
build-ready.

Deliver a documented, versioned assumption/evidence inventory linked to existing
invariants, dependency contracts and BUILD obligations. Each load-bearing
assumption needs an owner, affected scope, falsifying scenario, required evidence
layer, current subject-bound evidence (or an explicit unresolved state), and the
handoff it must satisfy. Integrate with existing attestations/external checkers;
do not create a second incompatible evidence framework. Add a readiness report
that separately reports model consistency, semantic review, executable dependency
proof and implementation acceptance. A bounded/selected gate run must identify
unchecked obligations; no blanket completeness claim from selected green checks.

Acceptance: a synthetic model can pass structural/formal checks while the report
still refuses readiness for a declared, unproved restore-safety assumption. A pure
comparison fixture cannot satisfy a native restore experiment. Changed dependency
assumptions or evidence subjects invalidate the affected readiness claim. Legacy
migration is explicit and does not silently reclassify historical evidence.
No promise that Machinery discovers every missing assumption automatically.

## 10. Require concrete failure traces in cross-boundary design reviews

Problem: repeated LLM prose reviews can share assumptions. Locally consistent
contracts can be impossible together: admission needing future commit evidence,
authorization confused with worker identity, or a database/cloud operation treated
as one atomic transaction. H2 also exposed incompatible uniqueness, retry and
erasure guarantees only when their interactions were worked through.

Deliver a portable review protocol and synthetic benchmark corpus. For each
selected high-risk workflow, reviewers must trace real actors, inputs, authority,
state stores, transaction boundaries, external effects, observable outcomes and
recovery after each boundary. Include restore rollback, crash after an external
effect but before acknowledgement, duplicate/concurrent requests, revocation
races, unavailable evidence, retained data and unrelated shared-key content.
Produce counterexample traces and unresolved premises, not just VALIDATED labels.
Add a public declarative producer/consumer ordering contract only if a bounded
prototype shows value; any automated cycle check must distinguish impossible
future-proof dependencies from valid iterative/recovery protocols. Avoid keyword
heuristics advertised as semantic verification.

Acceptance: independent reviewers, without the expected-answer file, identify
seeded restore resurrection, future-result authorization, premature erasure,
retention/key coupling and unreadable uniqueness-reservation defects. Include
correct near-neighbours to measure false positives. Record what was actually
reviewed; a delta approval must not imply whole-system approval. Tool validation
checks trace structure and references, not the truth of arbitrary prose.

## 11. Govern early executable experiments for architecture assumptions

Problem: dependency semantics were accepted at the design abstraction level and
only challenged much later. H2 needs evidence for restored-key denial, original
transaction outcome, external-effect uncertainty and recovery fencing; ordinary
unit tests or API documentation alone cannot establish these properties.

Deliver a lightweight experiment lane integrated with architecture adoption and
existing checker/evidence contracts. Bind an experiment to one critical assumption,
a disposable environment description, exact subject/dependency versions, explicit
expected observations and negative controls, and a retained evidence disposition.
Experiments may inform a source revision but never count as production GREEN,
locked RED completion or milestone acceptance. Explicitly support recording
"not runnable without owner authority" without fabricating proof or provisioning
resources. Use existing safety/custody mechanisms rather than a new test runner
unless a concrete missing capability is demonstrated.

Acceptance: a local synthetic database-plus-independent-key-service experiment
reproduces successful decryption after restoring a deleted wrapped-key row;
a corrected denial mechanism survives that restore and the relevant negative
controls. Missing/stale evidence blocks only the declared dependent adoption or
handoff. Fake adapters cannot be labelled native proof. No cloud purchase, secret,
privilege or destructive cleanup is implied by experiment registration.

## 12. Bound revision impact and expose design-only delivery drift

Problem: one source repair can require many downstream updates, making normal
derivation look like many newly discovered defects. Conversely, review loops can
expand into later milestones without delivering the current slice. H2 records
contain both genuine safety repairs and an explicitly redirected M4 authoring
exploration during M1 cleanup.

Deliver a revision-impact report using Machinery's public traceability model:
classify the initiating change as a discovered defect, owner decision, planned
implementation detail or deferred expansion; enumerate affected invariants,
contracts, generated artifacts, locked tests/sanctions, packets and milestone
obligations. Distinguish direct semantic changes from generated propagation, and
report newly introduced/retained/resolved findings without confusing raw findings
with normalized ratchet groups. Show the shortest dependency-closed candidate
increment and remaining gates where derivable; never invent a gate waiver or claim
an independently committable subset merely because files can be separated.

Acceptance: a one-invariant repair with many generated changes reports one causal
revision plus its impact; unchanged oracle IDs with changed contract meaning remain
visible. A later-milestone-only proposal is flagged for explicit scope disposition,
not automatically made a current blocker. The report distinguishes source progress,
locked RED, runnable GREEN and accepted/pushed delivery. It preserves existing
baselines, owner authority, locked-test sanctions and acceptance requirements.

Evidence for issue refinement: H2 design/DECISIONS.md and
.claude/plans/m1-s5/reviews/rev18-domain-checkpoint.md as of 2026-09-14.
Use sanitized synthetic reproductions in Machinery; do not copy private H2
runtime evidence, credentials or local custody material into its fixtures.

## 13. Clarify and verify contract-only records without lifecycle machines

Concrete H2 observation on installed v0.8.0, 2026-09-14: a new
ConfidentialErasureRecord has no status/lifecycle and an explicit ARCH placement
`(no machine: immutable append-only material record; contract spec in
machines/ConfidentialErasureRecord.matrix.md; actual source-host and custody
lifecycles remain separately governed)`. Its named-unit matrix follows existing
H2 contract-only records. G3 reports an orphan matrix; Gd reports missing owning
machine/oracle for each declared guard. Adding the explicit matrix path to the
waiver did not resolve it. Existing RetentionPolicy/RunScope contract-only files
are accepted in the same corpus. This is an observed authoring/documentation
mismatch to investigate, not a diagnosed private-engine defect.

Deliver a minimal synthetic reproduction using only public authoring contracts;
determine whether the missing piece is a documented declaration, a validator bug
or an unsupported format. Define ONE public, portable way to bind immutable
record/creation-time contracts, their invariant enforcement and falsifying tests
without inventing lifecycle states or an empty oracle. Reconcile G3/Gd/Gx/Gt and
packet projection against that contract, document migration and diagnostics, and
add compatibility tests for existing contract-only records. Do not reverse-engineer
private schemas into consumer designs or grandfather arbitrary orphan files.

Acceptance: a newly authored valid contract-only record passes the applicable
checks without a fake machine; a stale undeclared orphan still fails; unknown
invariant/source and stale references fail; clause tests retain explicit coverage
obligations without phantom transition IDs; semantic/runtime proof limits remain
visible. The minimal fixture must establish why previously existing records and
new records differed. H2 will retain its requirements/test obligations in BUILD's
documented named-unit/traceability tables while this is investigated; no H2 gate
or locked test is waived and no Machinery upgrade is required by this proposal.

## 14. Validate assembled bounded values and exact source-type composition

Concrete H2 observation, 2026-09-14: independently reviewed local row/source
products respected their individual bounds, but assembling the whole nine-row
retained transaction result produced a 57,772-byte synthetic case under a 40,960-byte
ceiling. Removing duplicated row views alone could not make it fit. Nested base64,
repeated native representations, source metadata and genuine late read witnesses
all contributed. This was a synthetic source serialization finding, not proof of
an admitted native transaction or a global maximum. Separately, a protocol union
extension accidentally allowed a specialized expectation inside a generic result,
and a proposed prior-source link required its own future registration reference.

Deliver an early composition experiment/check contract for bounded wire and local
result designs. Require one complete assembled value at the actual consuming root,
all physical effects and mandatory read witnesses, exact canonical encoding and
nested-encoding expansion, and explicit evidence for each chosen parameter bound.
Validate source-specific union pairings and distinguish prior references, same-result
local leaves and future/self references. Prefer a public bounded prototype over
inventing a universal schema compiler; report unsupported semantics plainly.
Lossless representation changes must reconstruct every original native field/byte,
retain all authority/currentness observations, and preserve existing limits.

Acceptance: sanitized synthetic fixtures reproduce individual-products-pass /
whole-value-overflows, an extra-wrapper depth overflow, an accidental cross-product
of generic and specialized protocol arms, and a prior-source self-reference cycle.
Correct near-neighbours must pass, including retained late witnesses and exact
acyclic local deduplication. Evidence identifies actual assembled bytes/depth,
parameter premises, unresolved native applicability and exact field reconstruction.
No truncated rows, raised budgets, assumed maximum provider values or shape-only
fixtures may be reported as runnable GREEN or native proof. Integrate the findings
with workstreams 9-12 so they are challenged before a build-ready handoff, not only
when late implementation needs a concrete serializer.

Additional bootstrap regression cases for this workstream: include the complete
signed work wrapper around a six-field readiness value; the nested form reaches
depth 9 and lossless flattening reaches 8. Include a plan that claims to bind a
future result but has no encoded result identity: require the exact preallocated
logical target/result fields and later equality to actual registered execution,
without demanding a native backend/xid before planning. Initial input-manifest
verification must resolve the independently current frozen source, while later
readback may resolve the known work record; using that future record for initial
verification is a cycle. Keep these synthetic structural cases separate from
authenticated source, transaction and native behavior evidence.

## 15. Track authoritative source replacement and stale reservations

Concrete H2 reproduction: an older wire-contract draft reserved PublicationManifest
and ReadbackReceipt as missing. A newer publication-authority draft defined both,
and the schema catalog already named that newer owner. Reviewers read the old
reservation and proposed additional schema work before the conductor reconciled
the catalog. This was stale source ownership, not a newly discovered product need.
No private Machinery implementation behavior is inferred from this example.

Workstream:

- Define a documented, opt-in source-ownership and supersession record for stable
  contract/type identifiers, pointing to exact authored definitions and the older
  reservation or definition they replace. Reuse existing public artifact contracts
  where possible; do not infer authority from filenames or latest timestamps.
- Check the closed declared set for duplicate active owners, dangling replacements,
  supersession cycles and stale missing-definition claims after explicit promotion.
  Keep value-shape closure separate from source authority, implementation, native
  adoption and acceptance: a defined type can still have unavailable constructors.
- Make review/execution packets include the authoritative definition and its current
  constructor restrictions. Old reservation text should be visibly historical or
  rejected as active input rather than silently competing with the new source.
- Provide migration diagnostics for existing multi-draft designs; never rewrite
  authored decisions, delete older evidence or auto-promote a proposal.

Acceptance examples: a synthetic old reservation plus explicit replacement yields
one active owner; a stale active missing claim is flagged; duplicate owners and
cycles fail; unresolved native authority remains unresolved after schema closure;
a packet cannot select a superseded definition as its current contract. Prove
these cases with public fixtures and actual packet/check integration. These checks
cannot establish that the chosen schema is semantically correct or that undeclared
requirements are complete.

## 16. Challenge invented prerequisites and incompatible proof reuse

Concrete H2 observation, 2026-09-14: a private publication proposal treated
permanent shutdown of future observations as a prerequisite to ordinary immutable
publication. That led to proposing a new closure registry. Tracing the actual
W/T/C consumers showed they require exact original bytes/readback/retention,
truthful inspected-cut history and complete current copy accounting; future
independently authorized observations remain possible. Stronger final erasure
requires its actual scoped fencing and copy disposition. The extra ordinary
registry was rejected before adoption. A separate bootstrap completion proposal
borrowed a ScopedWork closure whose receipt requires actual erasure Closing and
Barrier facts; routine provisioning cannot manufacture those facts merely to
satisfy a type. Both were proposal/review errors, not demonstrated engine defects.

Deliver a documented prerequisite-review contract tied to actual consumers:

- Every newly blocking source dependency identifies the concrete consumer claim,
  authoritative requirement, exact applicability preconditions and a failure
  trace showing why the dependency is necessary for that claim.
- Distinguish historical/original result, current readiness, invocation completion,
  and final scoped disposal. A stronger proof may not silently become a prerequisite
  to a weaker claim. Existing effects, uncertainty and later-copy duties still apply.
- Check explicit receipt/source-family compatibility before reuse: matching field
  shapes or names do not import the producer's authority, lifecycle preconditions
  or scope. Require a recorded disposition for unsupported reuse rather than
  automatically creating entities, stores or lifecycle transitions to make it fit.
- Expose rejected/invented prerequisites separately from discovered defects in the
  prior design and from missing implementation evidence. Connect this review to
  workstreams 12 and 15 so a private proposal cannot become an unquestioned blocker
  through repetition in progress records or packets.

Acceptance: sanitized positive/negative fixtures distinguish ordinary immutable
publication from final erasure, and ordinary native completion from an erasure-only
closure. The wrong receipt/prerequisite is challenged with its exact incompatible
precondition; a genuine unresolved original create or unaccounted selected copy
still blocks the claim that depends on it. No assertion, retention requirement,
current-denial check or acceptance gate is weakened. The tool reports where
semantic necessity needs human/conductor adjudication rather than claiming that
an automated type/reference check proves it. H2 progress 6.154 and the retained
source-scope adjudication provide refinement evidence; copy no private native
material into Machinery fixtures.

## 17. Hook-state store: caller binding breaks agent-seat handoff

Concrete H2 observation, 2026-09-19 (installed v0.8.0): the H2 conductor seat
rotated from a Claude Code agent (weekly quota exhausted) to an OpenCode agent on
the same machine, same checkout, same binary. Every governed tool call from the
new seat was denied with "machinery governance cannot access its durable
project-obligation store: durable hook state directory ... changed native
identity; refusing to accept a replacement store". Diagnosis evidence: the
store's `.store-identity` records `directory unix:1000011:1cf2ac2e`; `ls -di` of
the live store directory prints 485665838, exactly that inode, so the directory
is the original and not a replacement. The same binary, same payload, against a
different root created a fresh store and allowed the call. `machinery doctor`
reported the store healthy (476 entries) because it does not exercise the caller
binding the hook path enforces; the plugin's own failure guidance says "run
machinery doctor", which returned ok. Four defects:

1. The refusal message names the wrong cause: it frames an intact, original
   store as a rejected replacement. The real mismatch is in what the generation
   hash binds, a caller-shaped identity of the seat that created the store.
2. Doctor cannot see the failure class it is pointed at: integrity and entry
   counts pass while the hook denies every call, leaving contradictory guidance
   and no repair path.
3. No sanctioned seat transition: the installer supports three native targets
   and the skill states an identical contract across hosts, yet the expected
   operational flow, one agent seat handing a project to another on the same
   root, leaves governance hard-down for the new seat. The only remedy is
   hand-moving the private state directory, the exact action the error text
   says machinery refuses, and it silently discards the pending-obligation
   ledger the store exists to hold.
4. The identity model is undocumented at the seam: no readable artifact states
   what the durable store binds beyond the root path hash (store directory
   device:inode, a generation over caller inputs), so the failure is
   undiagnosable from the installed docs.

Deliver:

- Name the mismatched component in the error (for example: store bound to
  native host X, this caller is Y), and reserve "refusing to accept a
  replacement store" for an actual device:inode mismatch.
- A first-class transition (for example `machinery hook-state adopt --root
  <root>`): verify the existing store's integrity, re-bind it to the current
  caller, journal the handoff, and report any pending obligations in the old
  ledger (touched files not yet green) instead of dropping them.
- Doctor exercises the same binding check the hook enforces and, on failure,
  names the transition command rather than reporting ok.
- Document the identity model in the installed skill or CLI reference: what the
  durable store binds, why, and which transitions are sanctioned.

Acceptance: a synthetic one-root, two-caller reproduction (two native hosts, or
a native host plus the bare hook protocol from a shell): the second caller's
first call is denied with the component-naming message; doctor reports the
binding failure and names the transition; the transition re-binds, journals,
preserves and reports pending ledger obligations, and the second caller
proceeds without manual state-directory surgery. A genuinely replaced store
directory (new inode) still refuses. Existing single-seat flows are unchanged
and the fail-closed direction is preserved.

### Addendum (H2, 2026-09-19 to 2026-09-20): three more findings from the same incident

(a) The darwin native witness binds OS-volatile components. The store's
identity carries the device id and the inode generation (`st_gen`); on macOS
both move under an OS update or a restore with no tampering, which is what
fired here: the anti-tamper refusal on an untouched store whose inode was
proven identical. A witness that changes when nothing was touched is not a
tamper witness.

(b) The loss sentinel defeats the quarantine remedy. Quarantining the store
directory whole and letting the first governed call create a fresh store still
refuses with "missing after prior initialization", because a sentinel at the
home root (`~/.machinery-hook-state-<key>.initialized`) outlives the store by
design. The real clearing sequence was quarantine the store AND remove the
sentinel, discovered from source; no installed document names it.

(c) The design-inventory fingerprint races any process writing into the tree.
The shipped wall script logs beside itself, in-tree by default, so while a wall
ran the hook's fingerprint saw a growing log, failed closed, and denied EVERY
later tool call of the agent, including the kills that would have stopped the
writer: a deadlock only a human outside the seat could break. The workaround is
to run the script from a copy outside the tree; the fix belongs here: log
outside the governed tree, or tolerate known-transient paths, or scope the
fingerprint to the design inventory it actually guards.

## 18. Full-root current attestation: the row is the whole tree, and it moves with every commit

Concrete H2 observation, 2026-09-20 (v0.8.0), at the M2 milestone seal: the
design's own protocol re-attests `gt.conformance-test-shape` as `kind: current`
once the implementation is reviewed. A current row carries the full-root-v1
manifest: every regular file under the implementation root with mode, size and
hash, 3117 entries and about fifteen thousand lines of `attestations.yaml` for
a 3000-file tree, and the row is STALE the moment one file is added, removed
or changed anywhere in the root. Consequences met on the first day:

1. The developer checkout and the CI container never hold the same tree: the
   checkout carries ignored files (`.env.local`, editor and tool caches,
   provider state), the container drops paths the project's Dagger module
   ignores (here `.claude` and `.vault`, which git tracks). A row generated in
   either is STALE in the other. The project had to invent a "wall-shaped
   clone" (a fresh clone of the commit minus the container's ignore list) as
   the only tree both agree on, and move every `machinery check --impl` and
   every `machinery attest --impl` into it.
2. Every commit to the branch, including one that changes a line of a status
   ledger or a plan file, stales the row, so each commit ends with a
   regeneration and an amend: a fifteen thousand-line diff per commit for a
   judgment that did not change, and a history that grows by the manifest each
   time.
3. Nothing in the row says which files are the SUBJECT of the claim. For
   `gt.conformance-test-shape` the subject is the test suites under the
   conformance directory and the oracle registry; the manifest binds the
   README, the infrastructure module and the plans directory to the same
   judgment with the same weight.

Deliver:

- A scoped policy beside `full-root-v1`: the attestor names the subject globs
  (for example `test/conformance/**`, `test/support/**`, the oracle registry)
  plus the design covers, and the gate binds those files and only those; a file
  outside the scope changing does not stale the row. The policy name and the
  globs live in the row, so the scope is reviewable.
- An `--ignore` list for generation and checking that takes the same
  patterns a CI export ignores, so the checkout and the container can be made
  to agree without a clone.
- A compact manifest form (one line per entry, or a Merkle root per directory
  with the per-file lines in a sidecar the record references by hash), so the
  attestation record stays readable by a human beside the judgments it holds.

Acceptance: a current row scoped to a subject set survives a commit that
touches files outside the set; a change inside the set stales it; the
checkout and an exported tree with the same ignore list produce the same scope
hash; and the record's growth per regeneration is bounded by the subject set,
not the tree.

## H2 conductor experience, one full day at the M2 seal and the M3 tail (2026-09-20)

Written by the seat that ran the day, as the owner asked: where machinery
over-does, where it under-does, with the evidence. The day landed eight lanes,
sealed M2, ran two acceptance reviews and one specialist review gate over 183
commits, and regenerated the design record eleven times. Entries 19 to 26.

## 19. Gt credits a suite that skips itself (under-doing, the most important one)

Six suites run the real conversion reader as container tool steps and carry
`@moduletag :reader_image`, skipping themselves by name wherever the pinned
image is not served, which is every CI container. `Gt-tests` credits them by
static discovery of their oracle-id tokens and executes nothing, so the design
read those obligations as bound while no wall ever ran the tests. A
fresh-context acceptance review found it, not a gate. The project had to build
its own instrument (a content-hash record of the last green out-of-band run,
refused on drift; `tools/ci/reader_record_gate.sh`), which is a workaround for a
gap the gate should own.

Deliver: `Gt` reads the skip vocabulary of the test framework it credits
(module and test tags `skip`, `pending`, an `exclude` list in the runner
config, an environment-gated `@moduletag` whose condition is a literal false in
CI) and reports a credited id whose only test is skipped as UNRUN, a distinct
finding class from unbound; and an opt-in "out-of-band record" row type that
binds such a suite to a recorded run by content hash, so the workaround becomes
a machinery contract. Acceptance: a suite with `@moduletag skip:` credits
nothing; the same suite with a committed record binding its hash credits; the
record stales when the suite or its named sources change.

## 20. Nothing checks that a declared effect has a consumer (under-doing)

The M3 machines declare effects in named units (`holdBatchWatermark` answers a
held watermark, `recordParseMarker` answers markers, the Failing arm answers an
alert, `completeRun` answers a sealed manifest) and the persistence driver
carried a closed list of "report keys" that silently dropped anything not a
column. Four declared effects went nowhere for weeks; a fifth was added to the
closed list to silence the raise that would have said so. Every gate was green:
the oracle rows were bound, the guards falsified, the invariants enforced. The
review gate found it by reading.

Deliver: a design-side vocabulary for what a unit's effect is CARRIED BY (a
column, an outbox emission, a sink, a signal, another machine's action), owed
per named unit the way `CLAUSES{}` is owed per guard, and a gate that refuses
a unit whose declared effect names no carrier; with `--impl`, a check that
each carrier resolves to a code site the same way `Gt` resolves an oracle id.
Acceptance: the four H2 cases fail the gate at the design level (no carrier
named) and the fifth fails with `--impl` (a carrier named that nothing
implements).

## 21. A locked suite edited without a tag is invisible to every gate (under-doing)

Hard TDD says a LOCKED suite changes only under a recorded `AMEND-` tag. A
lane's wip commit changed one line in two LOCKED suites (a fixture tuple shape)
and landed through a green wall and a green sixteen-gate check; a review found
it nine days later. Locking lives in the suite's moduledoc and in the project's
red ledger, and nothing in machinery reads either.

Deliver: a `locked:` inventory (paths or a moduledoc marker the design names)
and a gate under `--impl` with `--commit`: a locked file whose bytes differ from
the anchor commit's without an `AMEND-` token in the diff's commit messages or
in `DECISIONS.md` dated after the anchor is a blocking finding. Acceptance: the
H2 case (89436e6b) fails; the same change with a dated DECISIONS entry naming
the tag passes.

## 22. Re-judgment is a ritual that grows without bound (over-doing)

Every prose edit to a design source stales four to ten attestation rows, and
the remedy is by hand each time: run `machinery attest <paths>`, paste hashes
into the right rows, prepend a `RE-JUDGED <date> ...` line to each note. The
H2 record now carries rows with a dozen RE-JUDGED lines, three of them
byte-identical duplicates from one day, and the judgment text almost never
changes because a one-line prose addition does not move a placement or a
guard. Eleven regenerations today, each three to five minutes of a conductor's
attention, for zero changed judgments.

Deliver: `machinery attest --rejudge <claim> --note "<text>"` that refreshes
every cover hash of that row from the tree and appends the dated line in one
step; covers scoped to a section or a stable id rather than a whole file for
the claims where that is meaningful (`g4.zero-context` over `BUILD.md` moves
on every dated paragraph, and the reviewer re-reads a paragraph, not the
file); and a lint that refuses a duplicate RE-JUDGED line. Acceptance: the
eleven refreshes of 2026-09-20 become eleven one-line commands, and a row's
note never carries the same dated line twice.

## 23. `--complete` at a seal reports the open milestones as errors, by design (over-doing)

The project's protocol runs `machinery check --complete` once at every seal
and records the result. On a ten-milestone design with two closed it prints
nine `G!-complete` ERRORs (the open milestones) plus the whole-design Gt
findings the ratchet accounts for, exits 1, and the number means nothing to
anyone: 479 today. Deliver: `--complete --milestone <id>` that asks the
final-handoff question of one milestone (its acceptance bound, its DoD ids
bound, its shards' claims attested current, no warnings inside its scope),
so a seal has a check that can be green. Acceptance: the M2 seal's run reads 0
blocking; the whole-design `--complete` keeps its meaning for the last
milestone.

## 24. Read-only git commands on generated files are refused (over-doing)

A lane needed `git show 78f6814c -- design/machines/Connector.oracle.md` to
read a withdrawn commit's oracle and was denied: "shell commands may not
reference protected output regardless of verb". Reading history is not
regeneration; the lane reached the bytes through `git cherry-pick -n` instead,
a heavier and less transparent path. Deliver: the hook admits read-only git
verbs (`show`, `diff`, `log`, `cat-file`, `ls-tree`) on protected paths and
refuses only the verbs that write. Acceptance: the denied command runs; `sed
-i` on the same path is still refused.

## 25. The check is nine times slower in a checkout than in a clone (over-doing, cost)

`machinery check --impl .` took 3 minutes 38 seconds in the developer checkout
and 33 seconds in a clean clone of the same commit, because the checkout holds
ignored trees (editor caches, provider state, a Go module cache) the check
hashes and walks. Deliver: honor the repository's ignore rules by default
under `--impl` (with `--no-ignore` to opt out), or read the tracked set from
git when the root is a repository. Acceptance: the checkout run within ten
percent of the clone run.

## 26. Two small ones

(a) `machinery attest --design --impl --claim ... --kind current` emits a row
with the generator's key order (`attestor` first) and four-space list indent;
merging it into a hand-kept record needed a script. Deliver `--merge-into
<file>` that replaces the claim's row in place. (b) The acceptance file grammar
(single-quoted scalars; every id with its own prefix) cost each of three
reviews one retry, and the YAML error names the sequence's first line rather
than the offending apostrophe. Deliver `machinery lint-acceptance <file>` with
the offending line, or accept block scalars.

What machinery got RIGHT today, so the list above is read in proportion: the
Ga binding refused an acceptance whose id carried the wrong prefix; Gv caught
every stale hash; the wall's warning assertion caught the retired plan-only
warning; the packet budgets and the slice map held through five design edits;
and a fresh-context review against a named commit, with the file replaced
wholesale each round, is the reason M3 is not sealed on evidence it does not
have.

## 27. Packet budgets: a slice at 99.7 percent fails on the next sentence anyone writes (2026-09-21, H2 M4 design edit)

The M4 design edit tripped `Gw-packet` on M1-S4 (598097 of 600000 bytes before the edit wrote a line)
because M1-S4 cited `section:BUILD/assess.md#9` and the M4-and-M9 partition lived in a subsection of 9.
The remedy the gate suggests (cite subsections) worked once the subsection was promoted a heading level,
but M1-S8 now sits at 599841 of 600000, one hundred and fifty-nine bytes of headroom, and every packet in
the corpus grew by a uniform 1676 bytes in that edit, including packets citing nothing the edit touched.
Two asks: (a) `Gw-packet` reports headroom per slice as a warning band (say under 2 percent) before it
fails, so a conductor sees the landmine before the sentence that trips it; (b) the packet's fixed
overhead (whatever every packet carries) is reported as its own line, because 1676 bytes across thirty-one
packets is a design-independent cost nobody chose. Also: a design-only run on a corpus whose
`gt.conformance-test-shape` is `kind: current` always reads one blocking finding (`GV_IMPL_REQUIRED`), so a
design lane can never be "0 blocking"; a `--design-only` reading that treats that one finding as
"owed to the impl review" would let a design lane's green mean green.

## 33. (2026-09-22, H2) Five design edits inside one milestone; four gap classes account for nearly all items

H2's M4 needed five design edits (rulings 123, 138, 163/168, 184, 216), every one of them triggered by an
implementation lane hitting a design that was gate-green and still incomplete. Classified over the ~45
items: (A) a machine's System-actor write action with NO authorization row (11 items); (B) a fact a
named-unit contract, guard or payload NAMES with no attribute in the model and no declared derived fact
(9 items); (C) a vocabulary the matrix calls closed and states only in prose (5 items); (D) two statements
of one fact that disagree or one that is missing where a twin exists: the event payload cells, the
section 9.1 milestone table versus a bound oracle, the residual mark resolving in two artifacts (4 items).
The rest are packet-budget splits (entry 27) and prose. Classes A, B, C and the payload half of D shipped
in 0.9.0 as Gx extensions. Retained here as the classification record.

## 37 (residual). Gap class D: twins that must agree, beyond payload cells

Delivered in 0.9.0: payload cells in the matrix versus the Architecture Contract. Not delivered: the
milestone/oracle table (BUILD.md 9.1) that says an oracle is unbound while a locked suite binds it, and a
residual mark whose id resolves in the policy layer OR the model. The general rule for the checker: where
the design states one fact in two artifacts by design (an embed, a payload, a milestone binding), the
second statement is either generated from the first (Ge-embed's model) or compared to it, and a prose
restatement is a finding. Proposal: extend Ge-embed's declared-embed mechanism to the milestone table's
oracle bindings (derive 9.1's "bound at" column from the slice map and the Gt-bound suites with --impl).
