---
id: MAC-6ylh
title: "Gu-surfaces: target-side surface ledger gate"
status: closed
priority: 1
type: feature
labels: [story, accepted]
created_at: 2026-08-29T23:49:02Z
created_by: ramirosalas
updated_at: 2026-08-30T00:19:57Z
content_hash: "sha256:beb1095c369046e5e93d37b7e3ea8368a5b913e0948c58f3ea7eee2751e5128d"
assignee: ramirosalas
closed_at: 2026-08-30T00:19:51Z
close_reason: "Accepted: Gu-surfaces target surface ledger gate lands exactly to spec. Full suite verified independently (go test ./... -json: 1177 pass/0 fail/1 skip matching proof exactly), targetsurface tests pass standalone, pvg verify clean (6 pre-existing hook.go stubs confirmed identical on origin/main), pvg gates PASS (no BLOCK), suite.go wiring matches spec (gu beside gs in activation table, run block, knownGateSet, default list, decomposed-parent narrowing), no mocks in integration-shaped tests, no push/tag/version bump confirmed via git. Both dispatcher-sanctioned deviations (gd restoration in 3 gate-vocab lists, zero-human-actor non-blocking note) verified as described and treated in-scope."
---

## Description
Implement Gu-surfaces: the target-side surface ledger gate. Origin: H2 root-cause finding 2026-08-30 (eight admin-gated acts survived ten deep reviews with no named interface; cause: the human-act-to-surface mapping had no artifact, so no gate and no closed review list). This is the forward twin of Gs (legacy/surface.yaml).

ARTIFACT: design/surfaces.yaml (target design). Strict schema, unknown keys are errors (mirror internal/gates/surface.go conventions exactly):
- surface_version: 1 (required), _comment optional, sources: optional list of strings (where the act list was enumerated from).
- acts: required list. Row keys: act (required: "Entity.action" or "knob:<key>"), actor (required), surface (required, non-empty free text naming the screen/admin command/API route/config release), milestone (optional), _comment (optional).
- deferrals: optional list. Row keys: act (required: "Entity.action", "knob:<key>", or "actor:<Name>" to defer one persona wholesale), reason (required, non-empty), _comment (optional).

GATE Gu-surfaces ("Gu-surfaces  target surface ledger"):
- Activation: file presence (HasTargetSurfaces), same convention as Gs/Gp: auto-runs when design/surfaces.yaml exists; an explicit --gate gu with no file errors on the missing artifact, never skips silently.
- Closed set: parse every *.modelith.yaml at the design root with the SAME loading approach the Gs target model uses, extended to read each action's actor: field. Obligated set = every action whose actor is present and not "System".
- Checks, all deterministic: (1) schema strict; (2) COMPLETENESS: every obligated action is covered by exactly one acts row (exact Entity.action match) or by a deferral (act-level, or actor-level via "actor:<Name>"); a missing action is an ERROR naming it; (3) RESOLUTION: an acts row in Entity.action shape must name an existing entity and action (dangling = ERROR) and its actor must equal the model action's actor (mismatch = ERROR: the ledger must not misattribute an act); (4) knob: rows resolve against nothing (open set by design) but duplicate act values anywhere are ERRORS; (5) an actor-level deferral must name an actor that actually appears in the model; (6) actions with NO actor field carry no obligation, but their count prints in the checked line so partial actor adoption stays visible.
- checked line prints: obligated actions, covered, deferred acts, deferred personas, knob rows, actorless actions.

WIRING: internal/gates/suite.go: add {"gu", HasTargetSurfaces} beside gs in both the activation table and the run block (gu runs after gs). Sweep the repo for every enumerated gate list that names gs (cmd help text, docs, skills/machinery/SKILL.md gate lists, .claude-plugin and .codex-plugin adapters, README) and add gu consistently.

TESTS (standard TDD, table-driven, mirror surface_test.go): happy complete ledger; missing obligated action; dangling Entity.action; actor mismatch; unknown key at each level; actor-level deferral covering a persona; duplicate act row; knob rows accepted and counted; absent file = gate skipped in default run; explicit gu with absent file = error; actorless actions counted not obligated. Run the full repo test suite and lint (make targets / golangci per .golangci.yml) green.

DOCS + SKILL (same commit): docs/target-surfaces.md, a short guide in the docs/surface-ledger.md voice: the artifact, the gate, the persona-walk authoring sweep. skills/machinery/SKILL.md: (a) add surfaces.yaml to the output layout listing; (b) add gu to the gate enumerations; (c) in the Phase 2 section add the PERSONA-WALK sweep: for every human persona in the glossary, walk their complete action list into named surfaces before Gate 2, author design/surfaces.yaml, and extend the phase-exit self-review coverage question to include it; (d) mirror any per-reference gate list that would otherwise go stale. Keep the skill's voice; no em dashes anywhere.

CONSTRAINTS: no version bump, no tag, no push (the maintainer will land more changes before releasing). Coverage floor 80 percent for the new code. Do not modify unrelated gates.

## Acceptance Criteria


## Design


## Notes
### CI/Test Results

(Formatting note added by PM-Acceptor: this heading anchors the "CI/Test Results:" section already recorded above under Implementation Evidence -- full suite 1177 passed / 0 failed / 1 skipped from commit 158ace6, race suite pass, golden corpus pass, lint 0 issues, per-function coverage above the 80 percent floor. No content added; heading added to satisfy pvg story verify-delivery's schema check.)

### AC Verification

(Formatting note added by PM-Acceptor: this heading anchors the "Acceptance-criteria verification" table already recorded above, mapping every story requirement to its implementation location and verifying test. No content added; heading added to satisfy pvg story verify-delivery's schema check.)


## nd_contract
status: accepted

### evidence
- PM closeout applied via pvg story accept on 2026-08-29.

### proof
- [x] Story closed after accepted label was applied.


## Implementation Evidence

Summary: Gu-surfaces, the target-side surface ledger gate, lands on branch `mac-6ylh-gu-surfaces` at commit 158ace6. New gate `internal/gates/targetsurface.go` with table-driven tests, wired into the suite activation table, the default gate list, the decomposed-parent narrowing, `RunSelected`, and the stop-time hook fallback; `docs/target-surfaces.md` and `skills/machinery/references/target-surfaces.md` written; `surfaces.yaml`, the persona-walk sweep, the self-review coverage question, and `gu` added to every gate enumeration across the repo. Not pushed, no tag, no version bump.

Commit SHA: 158ace64aafd4df3170171d384f1728bce3af0fe (158ace6)

Commands run:

- `go test ./... -count=1 -v` -> exit 0; 1177 passed, 0 failed, 1 skipped
- `go test -race -count=1 ./internal/gates/ ./internal/hook/ ./cmd/machinery` -> all ok
- `go test -count=1 -run TestGolden ./cmd/machinery` -> ok, the golden corpus did not drift
- `golangci-lint run --config .golangci.yml --timeout 5m` -> "0 issues."
- `gofmt -l cmd/ internal/` -> empty; `go vet ./...` -> clean; `go mod tidy` -> go.mod/go.sum unchanged
- `go build ./cmd/machinery` and `make build` -> clean
- `pvg verify <changed source and docs> --format text --include-tests` -> PASSED, 0 issues on my files
- `.bin/machinery check` on all 8 example design suites -> exit 0 each
- docs gate (preflight step 6): no em dash and no emoji anywhere in README.md, CONTRIBUTING.md, install.sh, skills/, agents/, docs/, examples/, commands/, adapters/, hooks/, Makefile, .github/

CI/Test Results:

- Full suite: 1177 passed / 0 failed / 1 skipped, exit 0, produced from 158ace6. Every package ok: cmd/machinery, internal/{alloy,checker,compose,experiments,formal,gates,hook,install,ir,lint,oracle,pack,refine,tla,version}.
- Race suite (gates, hook, cmd/machinery): pass.
- Golden corpus: pass, no re-capture needed. `gu` only enters the byte-pinned decomposed-parent narrowing note when `surfaces.yaml` exists, and no example carries one.
- Lint (golangci-lint at the version pinned in `.golangci-version`): 0 issues.
- Coverage of the new code, `internal/gates/targetsurface.go`, per function from `go tool cover -func`: HasTargetSurfaces 100, key 100, obligated 100, actorless 100, CheckTargetSurfaces 95.2, errf 100, checkKeys 100, record 100, validateRoot 94.7, validateActs 92.9, resolveAct 100, validateDeferrals 87.5, checkCompleteness 100, readTargetActModel 100, readFile 83.9. Above the 80 percent floor on every function.
- The single skipped test is `TestStableIDPrefixCollisionIsExtended` (internal/oracle/oracle_test.go:161, "no collision found in budget"), pre-existing on main and outside this story's blast radius. No test of mine is skipped or environment-gated.
- `pvg verify` reports 6 `[stub] return empty string` findings in `internal/hook/hook.go`. The identical 6 exist on main (verified by stashing this branch's diff and re-running); they are legitimate "no warning" returns, and my edit to that file is a 3-line activation block.

## Functional smoke test

Temp design with a minimal `domain.modelith.yaml` (two entities Tenant and Invoice; three actions: `Tenant.suspend` actor TenantAdmin, `Tenant.reap` actor System, `Invoice.void` with no actor) plus a `surfaces.yaml` covering the TenantAdmin action.

```
$ .bin/machinery check $D --gate gu
== Gu-surfaces  target surface ledger ==
  checked: 1 obligated actions, 1 covered, 0 deferred acts, 0 deferred personas, 0 knob rows, 1 actorless actions
  ok

0 blocking (ERROR/DRIFT) finding(s)          [exit 0]
```

The System action carries no obligation and the actorless action is counted but not obligated, exactly as specified. Negative lane, same fixture with `acts: []` and a DEFAULT run (no `--gate`), which also proves auto-activation on file presence:

```
== Gu-surfaces  target surface ledger ==
  ERROR  Tenant.suspend (actor TenantAdmin) is named by no acts row and no deferral; every act a person performs needs a named surface or an explicit deferral
  checked: 1 obligated actions, 0 covered, 0 deferred acts, 0 deferred personas, 0 knob rows, 1 actorless actions
                                              [exit 1]
```

## Acceptance-criteria verification

| Story requirement | Where | Verified by |
|---|---|---|
| ARTIFACT `design/surfaces.yaml`, strict schema, unknown keys are errors | targetsurface.go root/act/deferral key sets + checkKeys | "unknown root key", "unknown act key", "unknown deferral key" cases |
| `surface_version: 1` required; `_comment` optional; `sources` optional list of strings | validateRoot | "bad version", "sources not strings"; `_comment` in all three allowed key sets |
| `acts` required list; row keys act/actor/surface/milestone/_comment with act, actor, surface required | validateRoot + validateActs | "acts missing", "acts not a list", "row without an actor", "row without a surface", "empty milestone", "acts row is not a mapping" |
| `deferrals` optional list; row keys act/reason/_comment; reason required non-empty | validateRoot + validateDeferrals | "deferrals not a list", "deferral without a reason", "deferral row is not a mapping" |
| Activation by file presence (HasTargetSurfaces); auto-runs when the file exists | suite.go activation table + RunSelected; hook.go stop-time fallback | TestCheckTargetSurfacesClean (default Select + RunSelected), TestTargetSurfacesActivation/absent |
| Explicit `--gate gu` with no file errors on the missing artifact, never skips silently | CheckTargetSurfaces early error | TestTargetSurfacesActivation/explicit, plus the smoke test |
| Closed set from every `*.modelith.yaml` at the design root, using the Gs loading approach extended with `actor` | readTargetActModel / readFile (sortedGlobExt, kind+version check) | TestCheckTargetSurfacesShardedModel: a second root shard obligates, `legacy/domain.modelith.yaml` does not |
| Obligated = actor present and not "System" | targetActModel.obligated | TestCheckTargetSurfacesActorlessCarryNoObligation (Deal.expire System, Deal.archive actorless: neither reported) |
| (1) schema strict | checkKeys at all three levels | three unknown-key cases |
| (2) COMPLETENESS: exactly one acts row or a deferral (act-level or actor-level); a miss is an ERROR naming it | checkCompleteness | "missing obligated action" names Tenant.suspend; all three deferral lanes in TestCheckTargetSurfacesDeferrals |
| (3) RESOLUTION: dangling Entity.action is an ERROR; actor mismatch is an ERROR | resolveAct | "dangling entity", "dangling action", "actor mismatch", "actor named for an actorless action" |
| (4) `knob:` rows resolve against nothing; duplicate act values anywhere are ERRORS | validateActs knob branch + record() spanning acts and deferrals | "duplicate act row", "act both mapped and deferred", "duplicate knob row", "knob with no key"; knob acceptance and counting in the clean and deferral tests |
| (5) actor-level deferral must name an actor that appears in the model | validateDeferrals actor: branch | "deferral of an unknown persona" |
| (6) actorless actions carry no obligation, count prints | obligated() excludes them; actorless() counted | TestCheckTargetSurfacesActorlessCarryNoObligation |
| checked line prints obligated actions, covered, deferred acts, deferred personas, knob rows, actorless actions | CheckedExtra, six segments, zeros stay visible | exact-string assertions in the clean test, all three deferral cases, and the no-human-acts test |
| WIRING: `{"gu", HasTargetSurfaces}` beside gs in the activation table and the run block, gu after gs | suite.go: knownGateSet, default list `gm,gs,gu,...`, narrowing table, RunSelected | TestCheckTargetSurfacesClean; golden corpus unchanged |
| Sweep every enumerated gate list naming gs | cmd/machinery/check.go (Use + flag help), README.md (pipeline, gate table, ledger narrative), docs/{external-checkers,acceptance-gate,brownfield-team-guide,surface-ledger}.md, commands/{check,design}.md, skills/machinery/SKILL.md, skills/machinery/tools/README.md, skills/machinery/references/surface-ledger.md | grep sweep on the full vocabulary string and on "Gs-surface"; `.claude-plugin/plugin.json` and `.codex-plugin/plugin.json` carry NO gate list (checked; nothing to change) |
| TESTS: happy, missing, dangling, actor mismatch, unknown key at each level, actor-level deferral, duplicate, knobs counted, absent file skipped, explicit gu errors, actorless counted | internal/gates/targetsurface_test.go, table-driven: 7 top-level tests, 41 assertions counted by `go test -run TargetSurface -v` | all pass |
| DOCS: docs/target-surfaces.md in the surface-ledger.md voice (artifact, gate, persona-walk sweep) | docs/target-surfaces.md | written; no em dashes, no emoji |
| SKILL (a) surfaces.yaml in the output layout | SKILL.md output layout block | line added beside workspace.dsl |
| SKILL (b) gu in the gate enumerations | SKILL.md `--gate` list, activation prose, artifact-activated list `gm/gs/gu/gp/gi/gn/gb`, References gate suite, brownfield staged list | grep-verified |
| SKILL (c) Phase 2 PERSONA-WALK sweep, and the phase-exit self-review coverage question extended | SKILL.md Phase 2 (new paragraph before the adoption-closure paragraph) and self-review question (4) | written in the skill's voice |
| SKILL (d) mirror per-reference gate lists | new skills/machinery/references/target-surfaces.md; references/surface-ledger.md cross-links its forward twin; tools/README.md gate-suite prose | written |
| CONSTRAINTS: no version bump, no tag, no push; coverage floor 80 percent; do not modify unrelated gates | git log; plugin.json versions untouched at 0.4.1; coverage above; diff touches only suite.go wiring, the hook activation fallback, docs, and new files | verified |

## Judgment calls, flagged for cheap rejection

One deliberate extra beyond the letter of the story: the `--gate` vocabulary lists in `commands/check.md`, `skills/machinery/SKILL.md`, and `skills/machinery/tools/README.md` were already missing `gd` before this change. I was editing those exact strings to add `gu`, so I restored `gd` at the same time rather than leaving a knowingly wrong list behind.

Two calls inside the gate itself:

- A run whose model has zero human actors emits a non-blocking note ("no target-model action names a human actor ... add actor: to the model's actions to arm this gate") rather than passing silently. The story did not ask for it; an unarmed gate that reads as green is the exact failure shape this story exists to close, so it seemed worth one note line. Easy to remove.
- Act-level deferrals are shape-checked (`Entity.action`, `knob:<key>`, or `actor:<Name>`) but not resolved against the model, because the story scopes resolution to acts rows. A typo'd deferral is still caught: it discharges nothing, so the completeness check reports the real act by name.

## LEARNINGS

- The Gs gate was the right template but not a literal one. `readSurfaceTargetModel` reads only `domain.modelith.yaml`, while the story asked for every `*.modelith.yaml` at the design root, so the model reader is a deliberate sibling rather than a shared helper. A future consolidation should preserve the root-sweep behavior, not collapse back to the single-file one.
- Zero-suppression in `Gate.Count` is a real trap for a coverage gate: "0 covered" and "0 obligated" are the two most important numbers this gate can print, and both would have vanished from the checked line. `CheckedExtra` exists for exactly this and its doc comment says so; reading that comment before writing the counts prevented a silent-green bug.
- The gate-list sweep is wider than it looks, and the obvious grep is the wrong one. A bare `\bgs\b` search drowns in "gates", "args", and `VersionSkewNote`'s own parameter name. Searching for the full vocabulary string (`gm,gs,gp,...`) plus "Gs-surface" found every real enumeration in one pass and turned up three lists that had already gone stale on `gd`. Worth doing on every gate that lands rather than trusting the lists to stay current.
- The decomposed-parent narrowing note is byte-pinned in the golden corpus and is built from the same activation table the new gate is wired into. Adding `gu` was safe only because activation is artifact-gated and no example carries `surfaces.yaml`. A gate that activated unconditionally would have re-captured the corpus as a side effect, which should never happen incidentally.
- Environment note for the dispatcher: `pvg story deliver` is permitted from this session, but every nd comment/notes mutation (`nd comments add`, `pvg issues comment`, `pvg nd update --append-notes`) is refused by the pvg guard with "Dispatcher mode is active. Mutating nd commands must be delegated to a tracked production agent." This proof therefore could not be written into the story; it needs to be appended to MAC-6ylh's Notes from a tracked developer worktree so `pvg story verify-delivery` turns green.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-08-29.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## History
- 2026-08-29T23:49:33Z status: open -> in_progress
- 2026-08-29T23:49:33Z claimed by ramirosalas
- 2026-08-30T00:06:22Z status: in_progress -> in_progress
- 2026-08-30T00:19:51Z status: in_progress -> closed

## Links


## Comments
