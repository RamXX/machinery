"""Tests for oracle_gen: stable test identity and rendering."""
import copy
import re

import oracle_gen
from conftest import minimal_machine


def rows_of(text):
    out = {}
    for line in text.splitlines():
        m = re.match(r"\|\s*(T-\S+)\s*\|\s*(\S+)\s*\|\s*(\S+)\s*\|\s*(\S+)\s*\|", line)
        if m:
            out[m.group(2)] = line  # stable id -> row
    return out


def test_render_has_stable_and_sequential_ids():
    text = oracle_gen.render(minimal_machine(), "Widget.machine.json")
    assert "| test id | stable id |" in text
    rows = rows_of(text)
    assert len(rows) == 5
    assert all(re.match(r"WIDG-[0-9a-f]{6}", sid) for sid in rows)


def test_stable_ids_survive_unrelated_insertion():
    """The row-number fragility from the review: inserting a transition must not
    change the identity of every downstream test."""
    m1 = minimal_machine()
    before = rows_of(oracle_gen.render(m1, "w"))

    m2 = copy.deepcopy(m1)
    # insert a new event on Draft, lexically before the existing ones
    m2["states"]["Draft"]["on"]["archive"] = {"target": "Published"}
    after = rows_of(oracle_gen.render(m2, "w"))

    assert set(before) <= set(after)
    lost = set(after) - set(before)
    assert len(lost) == 1  # exactly the new transition


def test_stable_id_changes_when_stimulus_changes():
    m1 = minimal_machine()
    before = rows_of(oracle_gen.render(m1, "w"))
    m2 = copy.deepcopy(m1)
    m2["states"]["Draft"]["on"]["publish"][0]["guard"] = "guardCanShip"
    after = rows_of(oracle_gen.render(m2, "w"))
    assert set(before) != set(after)


def test_stable_id_constant_when_only_expectation_changes():
    """Same stimulus, new expectation: the test is modified, not replaced."""
    m1 = minimal_machine()
    before = rows_of(oracle_gen.render(m1, "w"))
    m2 = copy.deepcopy(m1)
    m2["states"]["persisting"]["invoke"]["onDone"]["actions"] = "commitAndLog"
    after = rows_of(oracle_gen.render(m2, "w"))
    assert set(before) == set(after)
    changed = [sid for sid in before if before[sid] != after[sid]]
    assert len(changed) == 1


def test_render_is_deterministic():
    m = minimal_machine()
    assert oracle_gen.render(m, "w") == oracle_gen.render(copy.deepcopy(m), "w")
