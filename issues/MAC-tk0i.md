---
id: MAC-tk0i
title: "Executable test assurance has no CLI surface in v0.7.0: no machinery tdd command, no --store/--assurance flags, design/assurance ignored"
status: open
priority: 1
type: bug
labels: [assurance, cli, docs-mismatch]
parent: MAC-ui8a
created_at: 2026-09-08T07:55:16Z
created_by: ramirosalas
updated_at: 2026-09-08T07:55:16Z
content_hash: "sha256:335ddb17bb9af7b9c6ed11e71ef6323f109ac179b3893d3b1efb772aac1ce0ca"
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


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
