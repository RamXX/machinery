# Demos: what each capability looks like, and the script that proves it

Each row is a capability a user can observe, the command to see it, and the end-to-end scripts
that hold it. Run one script with `make e2e-one S=<name>`; the scripts are in
`e2e/testdata/script/`.

| Capability | See it | Proven by |
|---|---|---|
| A project file, four stage outcomes, exit codes | `machinery config`, `machinery checks` | `config_load`, `checks_list`, `skeleton_*` |
| The design spine: lint, cross-references, generation | `machinery lint`, `machinery gen` | `model_*`, `machines_*`, `xref_*`, `risk_*`, `failure_modes_*`, `gen_*` |
| Tables mean what XState 5.33.4 executes | `machinery lint --stage xstate` | `xstate_*` |
| Static gates: trace, layers, end-to-end tests only, scope, ratchet | `machinery check --base <ref>` | `trace_*`, `arch_*`, `tests_*`, `scope_static`, `ratchet_*` |
| The proof loop: receipts bound to a commit, the owed ratchet | `machinery run`, `verify`, `owed sync`, `proof`, `status` | `run_*`, `verify_*`, `owed_*`, `proof_*`, `status_shop`, `receipt_helper` |
| Test quality by mutation probing | `machinery probe -- <command>` | `probe_*` |
| Shift-right testing and the one landing | `machinery affected`, `machinery landing` | `affected_*`, `landing_*` |
| Formal models that cannot quietly lie (real TLC, real Alloy) | `machinery formal check` | `formal_*` (12 scripts), `ratchet_spec_weakened` |
| The backlog bridge on a real bd database | `machinery plan --apply`, `waves`, `brief` | `backlog_adapter`, `plan_*`, `waves_*`, `contract_freeze`, `brief_roles` |
| Tier gates and the fresh verifier | `machinery predict`, `accept`, `story` | `story_routine`, `story_consequential`, `story_critical`, `proof_story`, `verify_story` |
| Adoption in an existing project | `machinery assess`, `machinery adopt --rehearse`, `machinery adopt` | `assess_*`, `adopt_*`, `bfix_builds`, `brownfield_walkthrough` |
| A design kept in the project's own formats | `import` and `trace.markers` in `machinery.yaml` | `import_*`, `adopt_import` |
| Evidence from an existing Go suite | `machinery run --adapter go-test-json` | `run_adapter_*`, `verify_adapter_*`, `ratchet_adapter`, `trace_adapter_layout` |
| A checker of your own, from design to proven obligation | `machinery checkers run` | `checkers_*` |
| Guides whose commands are executed | `README.md`, `docs/getting-started.md`, `docs/brownfield.md` | `docs_readme`, `docs_getting_started`, `brownfield_walkthrough` |
| Agent surfaces, and hooks that only call the tool | `skills/`, `agents/`, `hooks/`, `machinery hook` | `plugins_manifest`, `hook_*` |
| machinery checking itself | `make check` | `self_check` |
| The whole loop on a real service and a real Postgres | `make crm` | `examples/crm/DEMO.md` |

## Three demos to run by hand

**The formal loop** (needs Java and `make tools`):

```
make e2e-one S=formal_check_shop
```

It generates a TLA+ skeleton from the order machine and an Alloy skeleton from the Modelith
model, runs both tools, finds the double-charge race when the atomicity assumption is switched
off, writes the counterexample as a trace and an obligation, checks four mutations and two reach
predicates, and shows that the pure stage then passes with no Java on the machine.

**Adoption** (needs `bd`):

```
make e2e-one S=brownfield_walkthrough
```

It takes an existing Go project with unit tests and its own backlog from a read-only assessment
and a rehearsal to adoption, evidence from its own tests (a Go suite through the run adapter,
direct and through a wrapper, and a script that emits receipts), a plan beside its own stories
and a widened scope, executing `docs/brownfield.md` command by command.

**Tier gates** (needs `bd`, Java):

```
make e2e-one S=story_consequential
make e2e-one S=story_critical
```

A consequential story fails its gate without a prediction, when the verifier is the builder, and
when the prediction is for another commit; it passes in the right order with two actors. A
critical story also needs its formal model proven and its counterexample replayed.
