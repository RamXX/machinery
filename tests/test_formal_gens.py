"""Tests for the formal-layer generators: tla_gen models what the machine says
(loudly refusing what it cannot model), and refine_gen refuses a semantics file
that has drifted from the machine (the review's central formal-layer finding).
"""
import copy
import json
import os

import pytest
import yaml

import tla_gen
import refine_gen
from conftest import minimal_machine, GO_CRM, FULFILLMENT


def _write(tmp_path, m, name="Widget"):
    p = tmp_path / f"{name}.machine.json"
    p.write_text(json.dumps(m), encoding="utf-8")
    return str(p)


# --------------------------------- tla_gen ---------------------------------

def test_tla_has_assumptions_block(tmp_path):
    mid, tla, cfg = tla_gen.generate(_write(tmp_path, minimal_machine()))
    assert "ASSUMPTIONS" in tla
    assert "Guards are erased" in tla
    assert "Single machine instance" in tla


def test_tla_gives_each_retry_state_its_own_counter(tmp_path):
    """The old generator modeled only the FIRST retry loop; a second one became
    an unbounded loop. Payment-style machines have two."""
    m = minimal_machine()
    m["states"]["persisting"]["invoke"]["onError"] = [
        {"target": "persistRetry", "guard": "isLocked", "actions": "recordError"},
        {"target": "Draft", "actions": "recordFail"},
    ]
    m["states"]["persistRetry"] = {
        "always": [{"target": "Draft", "guard": "retriesExhausted"}],
        "after": {"backoff": {"target": "persisting", "actions": "inc"}},
    }
    m["states"]["Draft"]["on"]["poke"] = {"target": "pokeWait"}
    m["states"]["pokeWait"] = {
        "invoke": {"src": "poker",
                   "onDone": {"target": "Published"},
                   "onError": {"target": "pokeRetry"}},
        "after": {"pokeTimeout": {"target": "pokeRetry"}},
    }
    m["states"]["pokeRetry"] = {
        "always": [{"target": "Draft", "guard": "pokesExhausted"}],
        "after": {"pokeBackoff": {"target": "pokeWait", "actions": "incPoke"}},
    }
    mid, tla, cfg = tla_gen.generate(_write(tmp_path, m))
    assert "rc1" in tla and "rc2" in tla
    assert "RetryAgain_persistRetry" in tla
    assert "RetryAgain_pokeRetry" in tla


def test_tla_rejects_nested_states(tmp_path):
    m = minimal_machine()
    m["states"]["Wrapper"] = {"initial": "Inner", "states": {"Inner": {"type": "final"}}}
    m["states"]["Draft"]["on"]["wrap"] = {"target": "Wrapper"}
    with pytest.raises(SystemExit, match="nested states"):
        tla_gen.generate(_write(tmp_path, m))


def test_tla_models_multi_target_retry_resume(tmp_path):
    """DBLocked-style retries resume to one of several phases; the old
    generator silently dropped all but the first."""
    m = minimal_machine()
    m["states"]["Draft"]["on"]["lock"] = {"target": "locked"}
    m["states"]["locked"] = {
        "always": [{"target": "Draft", "guard": "retriesExhausted"}],
        "after": {"backoff": [
            {"target": "persisting", "guard": "phaseA", "actions": "inc"},
            {"target": "Draft", "guard": "phaseB", "actions": "inc"},
        ]},
    }
    mid, tla, cfg = tla_gen.generate(_write(tmp_path, m))
    assert 'st\' \\in {"Draft", "persisting"}' in tla


def test_tla_carries_exhaustive_notes(tmp_path):
    m = minimal_machine()
    m["states"]["router"] = {
        "always": [{"target": "Draft", "guard": "priorIsDraft"}],
        "_exhaustive": "prior ranges over {Draft} by construction",
    }
    m["states"]["Draft"]["on"]["route"] = {"target": "router"}
    mid, tla, cfg = tla_gen.generate(_write(tmp_path, m))
    assert "prior ranges over {Draft} by construction" in tla


# -------------------------------- refine_gen --------------------------------

DEAL_MACHINE = os.path.join(GO_CRM, "design", "machines", "Deal.machine.json")
DEAL_SEM = os.path.join(GO_CRM, "design", "formal", "Deal.semantics.yaml")
SAGA_MACHINE = os.path.join(FULFILLMENT, "design", "machines", "FulfillmentSaga.machine.json")
SAGA_SEM = os.path.join(FULFILLMENT, "design", "formal", "FulfillmentSaga.semantics.yaml")


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f) if path.endswith(".json") else yaml.safe_load(f)


def test_lifecycle_reconciles_and_emits():
    mid, files = refine_gen.emit_lifecycle(load(DEAL_MACHINE), load(DEAL_SEM),
                                           ("m", "s"))
    assert mid == "Deal"
    assert "RECONCILED against the machine" in files["DealData.tla"]


def test_lifecycle_rejects_stage_set_drift():
    """The review's core finding: edit the machine, the old generator stayed
    green. Removing a stage from the semantics must now fail."""
    sem = load(DEAL_SEM)
    sem["stages"] = ["Lead", "Qualified", "Proposal"]  # Negotiation dropped
    with pytest.raises(SystemExit, match="domain states disagree"):
        refine_gen.emit_lifecycle(load(DEAL_MACHINE), sem, ("m", "s"))


def test_lifecycle_rejects_machine_edit():
    """Add an advance from the terminal Won state (a real stage-forward
    violation the old pipeline could not see)."""
    m = load(DEAL_MACHINE)
    m["states"]["Won"]["on"]["advanceStage"] = [
        {"target": "persisting", "guard": "guardCanAdvance", "actions": "setPendingAdvance"}]
    with pytest.raises(SystemExit, match="must reject 'advanceStage'"):
        refine_gen.emit_lifecycle(m, load(DEAL_SEM), ("m", "s"))


def test_lifecycle_rejects_stale_rollback_routing():
    m = load(DEAL_MACHINE)
    m["states"]["rolledBack"]["always"] = m["states"]["rolledBack"]["always"][:-1]
    with pytest.raises(SystemExit, match="rollback routing"):
        refine_gen.emit_lifecycle(m, load(DEAL_SEM), ("m", "s"))


def test_lifecycle_requires_event_names():
    sem = load(DEAL_SEM)
    del sem["advance_event"]
    with pytest.raises(SystemExit, match="advance_event"):
        refine_gen.emit_lifecycle(load(DEAL_MACHINE), sem, ("m", "s"))


def test_saga_reconciles_and_models_partial_compensation():
    mid, files = refine_gen.emit_saga(load(SAGA_MACHINE), load(SAGA_SEM), ("m", "s"))
    tla = files["FulfillmentSagaData.tla"]
    assert "Undo_released" in tla and "Undo_refunded" in tla
    assert "PER OBLIGATION" in tla


def test_saga_rejects_step_order_drift():
    sem = load(SAGA_SEM)
    sem["states"] = ["Paying", "Reserving", "Shipping"]  # pay before reserve
    with pytest.raises(SystemExit):
        refine_gen.emit_saga(load(SAGA_MACHINE), sem, ("m", "s"))


def test_saga_rejects_machine_failure_route_drift():
    """The money bug shape: a later step whose failure path skips compensation."""
    m = load(SAGA_MACHINE)
    m["states"]["Paying"]["invoke"]["onError"]["target"] = "Failed"
    with pytest.raises(SystemExit, match="failure paths"):
        refine_gen.emit_saga(m, load(SAGA_SEM), ("m", "s"))


def test_saga_requires_undo_on_non_final_steps():
    sem = load(SAGA_SEM)
    del sem["obligations"]["Paying"]["undo"]
    with pytest.raises(SystemExit, match="compensating"):
        refine_gen.emit_saga(load(SAGA_MACHINE), sem, ("m", "s"))


# -------------------------------- compose_gen -------------------------------

import compose_gen

COMP = os.path.join(FULFILLMENT, "design", "formal", "checkout.composition.yaml")


def test_composition_validates_and_models_branching():
    name, tla, cfg = compose_gen.generate(load(COMP), load(SAGA_MACHINE),
                                          "FulfillmentSaga.machine.json")
    assert name == "Checkout"
    assert "Fail_Paying" in tla
    assert "Undo_payment" in tla and "Undo_reservation" in tla
    assert "CompensateStall" in tla
    assert "Inv_CleanCompensation" in cfg


def test_composition_rejects_step_order_drift():
    comp = load(COMP)
    comp["sequence"][0], comp["sequence"][1] = comp["sequence"][1], comp["sequence"][0]
    with pytest.raises(SystemExit, match="forward chain"):
        compose_gen.generate(comp, load(SAGA_MACHINE), "m")


def test_composition_rejects_coordinator_edit():
    """If the saga machine reroutes a failure path, the committed composition
    must fail to regenerate rather than keep proving the old ordering."""
    m = load(SAGA_MACHINE)
    m["states"]["Paying"]["invoke"]["onError"]["target"] = "Failed"
    m["states"]["Paying"]["after"]["payTimeout"]["target"] = "Failed"
    with pytest.raises(SystemExit, match="failure paths"):
        compose_gen.generate(load(COMP), m, "m")


def test_composition_requires_undo_on_non_final_steps():
    comp = load(COMP)
    del comp["sequence"][1]["undo"]
    with pytest.raises(SystemExit, match="undo"):
        compose_gen.generate(comp, load(SAGA_MACHINE), "m")
