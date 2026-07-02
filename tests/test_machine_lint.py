"""Tests for machine_lint: the shared IR and the structural lint.

Several cases are regression tests for review findings: silently skipped
XState constructs (state-level onDone, parameterized actions), vacuous
reconciliation, and the fully-guarded-always liveness hole.
"""
import copy
import glob
import json
import os

import machine_lint as ml
from conftest import minimal_machine


def errs_of(m, base="Widget.machine.json"):
    errs, warns, notes, counts = ml.lint_machine(m, base)
    return errs


def test_minimal_machine_is_clean(machine):
    errs, warns, notes, counts = ml.lint_machine(machine, "w")
    assert errs == []
    assert counts["states"] == 3
    assert counts["transitions"] == 5


def test_unknown_root_key_is_error(machine):
    machine["fancyExtension"] = True
    assert any("unsupported root key 'fancyExtension'" in e for e in errs_of(machine))


def test_unknown_state_key_is_error(machine):
    machine["states"]["Draft"]["onExit"] = "x"
    assert any("unsupported key 'onExit'" in e for e in errs_of(machine))


def test_parallel_state_is_error_not_silence(machine):
    machine["states"]["Draft"]["type"] = "parallel"
    assert any("parallel" in e for e in errs_of(machine))


def test_dangling_target_is_error(machine):
    machine["states"]["Draft"]["on"]["publish"][0]["target"] = "NoSuchState"
    assert any("dangling target" in e for e in errs_of(machine))


def test_state_level_ondone_is_seen(machine):
    """Review experiment I: a compound state's own onDone was invisible."""
    machine["states"]["Wrapper"] = {
        "initial": "Inner",
        "states": {"Inner": {"type": "final"}},
        "onDone": {"target": "NoSuchState", "actions": "ghostAction"},
    }
    machine["states"]["Draft"]["on"]["wrap"] = {"target": "Wrapper"}
    errs = errs_of(machine)
    assert any("dangling target 'NoSuchState'" in e for e in errs)


def test_parameterized_actions_are_seen(machine):
    machine["states"]["Draft"]["entry"] = [{"type": "announce"}]
    guards, actions, actors = ml.machine_unit_names(machine)
    assert "announce" in actions


def test_bogus_action_value_is_error(machine):
    machine["states"]["Draft"]["entry"] = [42]
    assert any("unsupported action value" in e for e in errs_of(machine))


def test_unreachable_state_is_error(machine):
    machine["states"]["Orphan"] = {"on": {"poke": {"target": "Draft"}}}
    assert any("unreachable state Orphan" in e for e in errs_of(machine))


def test_reachability_through_compound_initial(machine):
    machine["states"]["Wrapper"] = {
        "initial": "Inner",
        "states": {"Inner": {"on": {"go": {"target": "Deep"}}},
                   "Deep": {"type": "final"}},
    }
    machine["states"]["Draft"]["on"]["wrap"] = {"target": "Wrapper"}
    errs = errs_of(machine)
    assert not any("unreachable" in e for e in errs), errs


def test_dead_end_leaf_is_error(machine):
    machine["states"]["Draft"]["on"]["park"] = {"target": "Parked"}
    machine["states"]["Parked"] = {}
    assert any("dead-end non-final leaf state Parked" in e for e in errs_of(machine))


def test_invoke_without_onerror_is_error(machine):
    del machine["states"]["persisting"]["invoke"]["onError"]
    assert any("has no onError" in e for e in errs_of(machine))


def test_invoke_without_after_is_error(machine):
    del machine["states"]["persisting"]["after"]
    assert any("no after/timeout" in e for e in errs_of(machine))


def test_final_state_with_transitions_is_error(machine):
    machine["states"]["Published"]["on"] = {"poke": {"target": "Draft"}}
    assert any("final state Published declares transitions" in e for e in errs_of(machine))


def test_compound_without_initial_is_error(machine):
    machine["states"]["Wrapper"] = {"states": {"Inner": {"type": "final"}}}
    machine["states"]["Draft"]["on"]["wrap"] = {"target": "Wrapper"}
    assert any("compound state Wrapper has no initial" in e for e in errs_of(machine))


def test_shadowed_branch_is_error(machine):
    machine["states"]["Draft"]["on"]["publish"] = [
        {"target": "persisting", "actions": "setPending"},
        {"actions": "recordDenied"},
    ]
    assert any("unreachable" in e and "branch" in e for e in errs_of(machine))


def test_fully_guarded_always_without_escape_is_error(machine):
    machine["states"]["router"] = {
        "always": [{"target": "Draft", "guard": "priorIsDraft"}]
    }
    machine["states"]["Draft"]["on"]["route"] = {"target": "router"}
    errs = errs_of(machine)
    assert any("fully guarded always-list" in e for e in errs)


def test_exhaustive_annotation_discharges_guarded_always(machine):
    machine["states"]["router"] = {
        "always": [{"target": "Draft", "guard": "priorIsDraft"}],
        "_exhaustive": "priorStage ranges over {Draft} by construction",
    }
    machine["states"]["Draft"]["on"]["route"] = {"target": "router"}
    errs, warns, notes, counts = ml.lint_machine(machine, "w")
    assert not any("fully guarded" in e for e in errs)
    assert any("exhaustiveness" in n for n in notes)


def test_guarded_always_with_after_escape_is_fine(machine):
    machine["states"]["retry"] = {
        "always": [{"target": "Draft", "guard": "exhausted"}],
        "after": {"backoff": {"target": "persisting"}},
    }
    machine["states"]["Draft"]["on"]["r"] = {"target": "retry"}
    errs = errs_of(machine)
    assert not any("fully guarded" in e for e in errs)


def test_ambiguous_simple_target_is_error(machine):
    machine["states"]["A"] = {
        "initial": "Dup", "states": {"Dup": {"type": "final"}},
        "on": {"x": {"target": "Dup"}},
    }
    machine["states"]["B"] = {
        "initial": "Dup", "states": {"Dup": {"type": "final"}},
    }
    machine["states"]["Draft"]["on"]["a"] = {"target": "A"}
    machine["states"]["Draft"]["on"]["b"] = {"target": "B"}
    assert any("ambiguous target" in e for e in errs_of(machine))


def test_bad_initial_is_error(machine):
    machine["initial"] = "Nowhere"
    assert any("initial 'Nowhere'" in e for e in errs_of(machine))


def test_load_machine_reports_json_error(tmp_path):
    p = tmp_path / "Bad.machine.json"
    p.write_text("{not json", encoding="utf-8")
    m, err = ml.load_machine(str(p))
    assert m is None and "invalid JSON" in err


# ---------------------------- matrix reconciliation ------------------------

MATRIX_OK = """
## Transition matrix

| # | source | event / after / always | guard | target | actions |
|---|---|---|---|---|---|
| 1 | Draft | publish | guardCanPublish | persisting | setPending |
| 2 | Draft | publish | !guardCanPublish | Draft (internal) | recordDenied |
| 3 | persisting | invoke onDone | - | Published | commit |
| 4 | persisting | invoke onError | - | Draft | recordError |
| 5 | persisting | after persistTimeout | - | Draft | recordTimeout |
"""


def test_matching_matrix_reconciles(machine):
    drift, n = ml.reconcile_matrix(machine, MATRIX_OK, "w")
    assert drift == []
    assert n == 5


def test_retargeted_transition_is_drift(machine):
    """Review experiment D: retargeting a transition must not pass."""
    machine["states"]["persisting"]["invoke"]["onDone"]["target"] = "Draft"
    drift, _ = ml.reconcile_matrix(machine, MATRIX_OK, "w")
    assert any("machine transition has no matrix row" in d for d in drift)
    assert any("matrix row has no machine transition" in d for d in drift)


def test_moved_action_is_drift(machine):
    """Review experiment C: swapping an action between transitions must not pass."""
    machine["states"]["Draft"]["on"]["publish"][0]["actions"] = "recordDenied"
    drift, _ = ml.reconcile_matrix(machine, MATRIX_OK, "w")
    assert drift


def test_renamed_guard_is_drift(machine):
    """Review experiment J: renaming a guard must not pass."""
    machine["states"]["Draft"]["on"]["publish"][0]["guard"] = "guardCanPublsh"
    drift, _ = ml.reconcile_matrix(machine, MATRIX_OK, "w")
    assert drift


def test_extra_matrix_row_is_drift(machine):
    extra = MATRIX_OK + "| 6 | Draft | archive | - | Published | archiveIt |\n"
    drift, _ = ml.reconcile_matrix(machine, extra, "w")
    assert any("matrix row has no machine transition" in d for d in drift)


def test_matrix_without_transition_table_is_not_drift(machine):
    text = "## Named units\n\n| name | kind | signature |\n|---|---|---|\n| `x` | guard | f |\n"
    drift, n = ml.reconcile_matrix(machine, text, "w")
    assert drift == [] and n == 0


def test_namedunit_names_parses_backticked_groups():
    text = ("| name | kind | signature | pre / post | maps to |\n"
            "|---|---|---|---|---|\n"
            "| `guardA` / `guardB` | guard | f | p | inv-x |\n"
            "| plainName | action | g | q | - |\n")
    names = ml.namedunit_names(text)
    assert {"guardA", "guardB", "plainName"} <= names


# ------------------------------ integration -------------------------------

def test_go_crm_machines_reconcile_with_their_matrices(go_crm_design):
    """The shipped example must satisfy the strict structural reconciliation."""
    mdir = os.path.join(go_crm_design, "machines")
    for path in sorted(glob.glob(os.path.join(mdir, "*.machine.json"))):
        m, err = ml.load_machine(path)
        assert err is None
        mx = path.replace(".machine.json", ".matrix.md")
        if not os.path.exists(mx):
            continue
        with open(mx, encoding="utf-8") as f:
            drift, n = ml.reconcile_matrix(m, f.read(), os.path.basename(path))
        assert drift == [], drift
        assert n > 0


def test_go_crm_machines_lint_clean(go_crm_design):
    mdir = os.path.join(go_crm_design, "machines")
    files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    assert files
    for path in files:
        m, err = ml.load_machine(path)
        assert err is None
        errs, warns, notes, counts = ml.lint_machine(m, os.path.basename(path))
        assert errs == [], errs


def test_fulfillment_machine_lints_clean(fulfillment_design):
    mdir = os.path.join(fulfillment_design, "machines")
    for path in sorted(glob.glob(os.path.join(mdir, "*.machine.json"))):
        m, err = ml.load_machine(path)
        assert err is None
        errs, warns, notes, counts = ml.lint_machine(m, os.path.basename(path))
        assert errs == [], errs


# --------------------------- event completeness ----------------------------

def test_resting_state_missing_event_is_error(machine):
    machine["states"]["Parked"] = {"on": {"unpark": {"target": "Draft"}}}
    machine["states"]["Draft"]["on"]["park"] = {"target": "Parked"}
    errs = errs_of(machine)
    assert any("Parked neither handles nor explicitly ignores event 'publish'" in e
               for e in errs)
    assert any("Draft neither handles nor explicitly ignores event 'unpark'" in e
               for e in errs)


def test_ignores_notation_discharges_completeness(machine):
    machine["states"]["Parked"] = {
        "on": {"unpark": {"target": "Draft"}},
        "_ignores": {"publish": "a parked widget cannot be published; unpark first",
                     "park": "already parked; idempotent no-op"},
    }
    machine["states"]["Draft"]["on"]["park"] = {"target": "Parked"}
    machine["states"]["Draft"]["_ignores"] = {"unpark": "not parked; nothing to do"}
    assert not any("neither handles" in e for e in errs_of(machine))


def test_ignores_requires_reason_strings(machine):
    machine["states"]["Draft"]["_ignores"] = {"publish": ""}
    assert any("_ignores must map event names to reason strings" in e
               for e in errs_of(machine))


def test_handling_and_ignoring_same_event_is_error(machine):
    machine["states"]["Draft"]["_ignores"] = {"publish": "never mind"}
    assert any("both handles and ignores event 'publish'" in e for e in errs_of(machine))


def test_transient_states_are_exempt_from_completeness(machine):
    # persisting has an invoke: it must not be required to handle 'publish'
    assert not any("persisting neither handles" in e for e in errs_of(machine))
