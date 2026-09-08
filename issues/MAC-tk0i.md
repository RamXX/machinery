---
id: MAC-tk0i
title: "Executable test assurance has no CLI surface in v0.7.0: no machinery tdd command, no --store/--assurance flags, design/assurance ignored"
status: closed
priority: 1
type: bug
labels: [assurance, cli, docs-mismatch]
parent: MAC-ui8a
created_at: 2026-09-08T07:55:16Z
created_by: ramirosalas
updated_at: 2026-09-08T09:27:50Z
content_hash: "sha256:534282dcd66ead8db0bfb0911174cd7830492bc7b0709f215ecd30bbd1b4aded"
closed_at: 2026-09-08T09:27:50Z
close_reason: "Landed on main (release 0.7.0 line) at 5c35463: f9eef0b implementation status stated in the contract, CHANGELOG and README doc index"
---

## Description
## Symptom

The v0.7.0 CHANGELOG and docs/test-assurance-contract.md describe executable test assurance,
four native adapters and replay-input retention as shipped consumer capability. The shipped
v0.7.0 binary exposes none of it. `internal/tdd` and `internal/tdd/adapters` exist and are
imported by `internal/gates`, `internal/assuranceflow` and `internal/runtimeclosure`, but no
cobra command registers them and no gate reads a declaration.

## Reproduction

Binary: v0.7.0 release candidate.

```
$ machinery tdd --help
unknown command "tdd" for "machinery"

$ machinery check design --assurance strict
unknown flag: --assurance

$ machinery check design --store /tmp/x
unknown flag: --store

$ strings $(which machinery) | grep -c 'machinery.tdd'
0
```

Declaration is ignored entirely, not merely unenforced:

```
$ mkdir -p design/assurance && echo '{"schema":"machinery.tdd.plan/v1"}' > design/assurance/plan.json
$ machinery check design 2>&1 | grep -i 'assurance\|tdd\|gtd'
(no output)
```

A schema-invalid plan.json produces no diagnostic, no gate, no change in finding count.

Registration site: cmd/machinery/main.go:73-101 registers 27 commands; there is no `newTddCmd()`.

## Expected, per the shipped docs

docs/test-assurance-contract.md:297-309 lists as public commands:

```
machinery tdd store init | export | import
machinery tdd scaffold | capture | register | red | green | status | verify
machinery check <design> --impl <path> --store <path> --assurance strict
machinery check <design> --impl <path> --store <path> --complete
```

CHANGELOG 0.7.0 "Added" states "A design may declare `design/assurance/plan.json` and per-milestone
manifests" and "Capture publishes the exact subject, dependency, frozen, and design payload inputs
as content-addressed blobs in a private assurance store".

docs/test-assurance-contract.md:59 further states "Ordinary check/hooks activate cheap gtd when
BUILD, assurance controls, configured external store or an assurance declaration exists".

## Actual

None of the above is reachable or observed. A consumer following the contract document writes
`design/assurance/plan.json` and gets silence.

## Ask

Either wire the CLI surface described in the contract, or mark the contract and the CHANGELOG
entries as design-only until it lands. docs/release-notes.md rule 5 already states the principle
("Publication gating is stated as it is, not as planned"); the same rule should govern this entry.
Silence on a declared but unread control file is the failure mode this repo exists to prevent.

## H2 impact

MAJOR, and it decides a milestone plan. H2 (Elixir/Phoenix/Ash) is starting milestone 1 under hard
TDD with 1,700+ oracle stable ids owed tests. The `elixir-exunit/v1` adapter
(docs/test-assurance-contract.md:279-289, Elixir 1.20.4 / OTP 29) is exactly the mechanism its
build plan needs to bind ExUnit tests to oracle rows. On the shipped binary H2 must instead fall
back to Gt-tests string matching of stable ids in test files, which the same release explicitly
downgraded to "static discovery; tests not executed". H2 needs to know whether to plan for the
adapter in M1 or to design around its absence.

## Acceptance Criteria


## Design


## Notes
Verified at release commit d1f2264 in the main checkout: cmd/machinery/main.go still registers no newTddCmd (grep -c newTddCmd = 0), and skills/machinery/SKILL.md still has no tdd/assurance/elixir-exunit/recover mention.

## History
- 2026-09-08T09:27:50Z status: open -> closed

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-08T09:15:13Z ramirosalas
Fixed on fix/MAC-3iga-wcrd at 38d0241.

Confirmed the report against a fresh build (make build; .bin/machinery --help; check --help): no tdd command, check accepts only --impl/--commit/--gate/--complete/--warnings-as-errors, and gates.AssuranceInventory is called only from internal/assuranceflow, so no gate in the check suite reads design/assurance/. Rather than claim an unimplemented CLI, the docs now state the split.

Shipped in 0.7.0: the closed version-1 document grammar and its validation (LoadPlan, LoadManifest, Validate, the review projections and subject digests, ValidateControlNamespace); gates.AssuranceInventory over a held design snapshot; the content-addressed store and replay-input retention (InitStore, Capture, MaterializeBundle, ExportStore, ImportStore, Status); assuranceflow.Register; the four closed adapters go-testing/v1, node-test-typescript/v1, python-unittest/v1, elixir-exunit/v1; and their execution in 'go run ./scripts/integration-lane --lane required', which verifies each runtime against testdata/integration-lanes/assurance-runtime-pins.json and runs frozen per-language probe and native-conformance fixtures under process custody.

Not shipped: every machinery tdd command (scaffold, red and green have no implementation at all); check --store and --assurance strict; any gate reading design/assurance/, and the section 3 gtd activation; Execute/Verify replay over a consumer design; and adoption of a design's tests by the shipped gates.

Changes: (a) docs/test-assurance-contract.md gains an 'Implementation status in 0.7.0' section at the top listing both lists and marking section 7's CLI grammar as the target contract; the contract itself is untouched. (b) The CHANGELOG 0.7.0 'Executable assurance declarations and four native test adapters' entry is rewritten to claim only what the binary does, and the capture and registration entries are marked library surfaces. (c) README carried no false tdd/replay/plan.json claim (its residual-obligation paragraph already said 'until the standalone test-assurance contract's enforcement lanes land'), so the fix there was the omission: the doc index now lists both architecture contracts with the same status.

For a consumer today: bind tests to oracle rows with Gt-tests stable ids; a committed plan.json changes nothing machinery check reports.

Verified: go test -count=1 -run 'AssuranceDocs|Docs|Readme|Contract|Skill' ./cmd/machinery passes; em-dash and toolchain doc scans clean.
