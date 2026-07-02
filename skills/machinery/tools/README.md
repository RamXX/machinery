# machinery check: deterministic verification gates

Pure static analysis over a machinery design (and, with `--impl`, the generated code).
No LLM. These are the hard symbolic checks that make correctness not depend on an agent
getting every cross-reference right. This is the "modelith lint" idea extended to the how
(C4) and the behavior (machines), plus the cross-layer seams.

```sh
python3 machinery_check.py <design-dir> [--impl <code-dir>]
```

Exit is non-zero on any ERROR or DRIFT. `warn` and `note` do not fail the gate.

## Gates

| Gate | What it checks | Kills |
|---|---|---|
| G2-c4 | The Architecture Contract parses; boundary ids are unique and map to `workspace.dsl` elements; `allow`/`deny` reference declared boundaries and never contradict; every external dependency has a mitigation row. | drift between the C4 model and its contract |
| G3-machine | Each machine is well-formed over its finite graph: every transition target resolves, no dead-end non-final state, every `invoke` has both `onError` and an `after` timeout. Then oracle reconciliation: every transition action appears in that transition's matrix row, and every entry/exit action is declared in the named-unit contract table. | malformed machines; machine-vs-oracle drift (defect class 2) |
| Gx-trace | Cross-layer: each Deal/Task/User machine's domain states are `DealStage`/`TaskStatus`/`UserStatus` values (overlay states are noted), each domain event is a Modelith action, and every invariant id is enforced somewhere (a matrix maps-to cell or BUILD.md section 6). | silent drift between the what, the how, and the behavior |
| G4-import | (needs `--impl`) Maps each source file to a contract boundary via the `code` globs, extracts imports, and asserts no cross-boundary import violates `allow`/`deny`. A specific `allow` overrides a wildcard `deny`. | boundary erosion in code (generalizes the single C-ARCH-01 test to every boundary) |

## The machine IR

`machine_lint.walk_states` / `transitions_of` / `actions_of` parse a machine JSON into a
reusable intermediate representation: states (with nesting), transitions tagged by kind
(`on` / `after` / `always` / `onDone` / `onError`) with guard, target, and actions, plus
entry/exit actions and invokes. The same IR is what the planned TLA+/NuSMV generator will
consume to model-check safety and liveness, and what the recursion layer will use for the
assume-guarantee composition. Keeping one IR means the deterministic gates and the coming
formal layer never diverge on how a machine is read.

## Design principle: generate, do not co-author

Where an artifact can be derived it should be generated, not authored twice. The transition
matrix and the test oracle are derivable from the machine JSON; generating them makes the
G3 drift class impossible by construction rather than merely detected. G3 today detects the
drift; the oracle generator (next) removes it. `machine_lint.py` is the standalone structural
linter; `machinery_check.py` is the full suite and supersedes its drift check.
