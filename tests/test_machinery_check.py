"""Tests for machinery_check: every gate must fail loudly on absence and on the
review's mutation experiments (A, B, D, E, F1, G, H), and pass on the coherent
synthetic design.
"""
import json
import os
import re
import shutil

import pytest

import machinery_check as mc
from fixtures import write_design, write_impl


@pytest.fixture
def design(tmp_path):
    return write_design(str(tmp_path))


@pytest.fixture
def impl(tmp_path):
    return write_impl(str(tmp_path))


def _edit(path, old, new):
    text = open(path, encoding="utf-8").read()
    assert old in text, f"fixture drift: {old!r} not in {path}"
    with open(path, "w", encoding="utf-8") as f:
        f.write(text.replace(old, new))


# ------------------------------ baseline ----------------------------------

def test_synthetic_design_passes_all_gates(design, impl):
    for gate in (mc.check_c4(design), mc.check_machines(design),
                 mc.check_traceability(design), mc.check_imports(design, impl)):
        assert gate.errs == [], (gate.title, gate.errs)
        assert gate.drift == [], (gate.title, gate.drift)


def test_gates_report_what_they_checked(design, impl):
    g2 = mc.check_c4(design)
    assert g2.counts["boundaries"] == 2
    assert g2.counts["dsl elements"] >= 3
    assert g2.counts["dependencies with mitigation rows"] >= 1
    g3 = mc.check_machines(design)
    assert g3.counts["machines"] == 1
    assert g3.counts["transitions"] == 5
    assert g3.counts["oracles fresh"] == 1
    gx = mc.check_traceability(design)
    assert gx.counts["lifecycle machines traced"] == 1
    assert gx.counts["invariants enforced"] == 1
    g4 = mc.check_imports(design, impl)
    assert g4.counts["go files checked"] == 2
    assert g4.counts["edges verified"] >= 2


# ----------------------- experiment A: empty design ------------------------

def test_empty_design_fails_every_gate(tmp_path):
    """Review experiment A: a nearly-empty design passed all gates. Never again."""
    d = tmp_path / "design"
    d.mkdir()
    (d / "ARCHITECTURE.md").write_text(
        "## 4. Architecture Contract\n\n```yaml\ncontract_version: 1\n```\n",
        encoding="utf-8")
    assert mc.check_c4(str(d)).errs
    assert mc.check_machines(str(d)).errs
    assert mc.check_traceability(str(d)).errs


def test_missing_impl_dir_is_error(design):
    g = mc.check_imports(design, "/nonexistent/impl")
    assert g.errs


def test_impl_with_no_source_is_error(design, tmp_path):
    empty = tmp_path / "emptyimpl"
    empty.mkdir()
    g = mc.check_imports(design, str(empty))
    assert any("checked nothing" in e or "no imports" in e or "no source files" in e
               for e in g.errs)


# --------------------- experiment B: mitigation table ----------------------

def test_deleted_mitigation_table_is_error(design):
    """Review experiment B: deleting the whole mitigation table passed G2."""
    arch = os.path.join(design, "ARCHITECTURE.md")
    text = open(arch, encoding="utf-8").read()
    start = text.index("## 6.")
    end = text.index("## 7.")
    open(arch, "w", encoding="utf-8").write(text[:start] + text[end:])
    g = mc.check_c4(design)
    assert any("no mitigation" in e or "mitigation row" in e for e in g.errs)


def test_unknown_dependency_in_mitigation_table_is_error(design):
    _edit(os.path.join(design, "ARCHITECTURE.md"), "| `db` |", "| `dbz` |")
    g = mc.check_c4(design)
    assert any("`dbz`" in e for e in g.errs)


# --------------------- experiment H: contract locator ----------------------

def test_contract_found_despite_earlier_yaml_block(design):
    """Review experiment H: the first yaml block used to win. The fixture has a
    decoy `replicas: 3` block before the contract; the contract must still load."""
    g = mc.check_c4(design)
    assert g.counts["boundaries"] == 2


def test_duplicate_boundary_id_is_error(design):
    _edit(os.path.join(design, "ARCHITECTURE.md"),
          "  - id: widget.store", "  - id: widget.app")
    g = mc.check_c4(design)
    assert any("duplicate boundary id" in e for e in g.errs)


def test_edge_both_allowed_and_denied_is_error(design):
    _edit(os.path.join(design, "ARCHITECTURE.md"),
          '- "widget.app -> external.db"', '- "widget.app -> widget.store"')
    g = mc.check_c4(design)
    assert any("both allowed and denied" in e for e in g.errs)


def test_rule_referencing_undeclared_boundary_is_error(design):
    _edit(os.path.join(design, "ARCHITECTURE.md"),
          "- widget.app -> widget.store", "- widget.app -> widget.ghost")
    g = mc.check_c4(design)
    assert any("undeclared boundary 'widget.ghost'" in e for e in g.errs)


def test_missing_workspace_dsl_is_error(design):
    os.remove(os.path.join(design, "workspace.dsl"))
    g = mc.check_c4(design)
    assert any("workspace.dsl" in e for e in g.errs)


def test_boundary_without_dsl_element_is_error(design):
    _edit(os.path.join(design, "workspace.dsl"),
          'storelib = component "Store"', 'storelibX = component "Store"')
    g = mc.check_c4(design)
    assert any("maps to no workspace.dsl element" in e for e in g.errs)


# ------------------- experiments D/E: machines and oracle -------------------

def test_stale_oracle_is_drift(design):
    """Review experiment E: a stale committed oracle passed with a message
    claiming drift was impossible."""
    mp = os.path.join(design, "machines", "Widget.machine.json")
    m = json.load(open(mp, encoding="utf-8"))
    m["states"]["Draft"]["on"]["publish"][0]["actions"] = "setPendingRenamed"
    json.dump(m, open(mp, "w", encoding="utf-8"), indent=1)
    g = mc.check_machines(design)
    assert any("stale" in d for d in g.drift)


def test_missing_oracle_is_error(design):
    os.remove(os.path.join(design, "machines", "Widget.oracle.md"))
    g = mc.check_machines(design)
    assert any("no committed oracle" in e for e in g.errs)


def test_retargeted_transition_is_matrix_drift(design):
    """Review experiment D."""
    mp = os.path.join(design, "machines", "Widget.machine.json")
    m = json.load(open(mp, encoding="utf-8"))
    m["states"]["persisting"]["invoke"]["onDone"]["target"] = "Draft"
    json.dump(m, open(mp, "w", encoding="utf-8"), indent=1)
    import oracle_gen
    open(os.path.join(design, "machines", "Widget.oracle.md"), "w",
         encoding="utf-8").write(oracle_gen.generate(mp))
    g = mc.check_machines(design)
    assert any("no matrix row" in d for d in g.drift)


def test_unit_without_namedunit_row_is_drift(design):
    _edit(os.path.join(design, "machines", "Widget.matrix.md"),
          "| `guardCanPublish` | guard |", "| `guardCanPublishX` | guard |")
    g = mc.check_machines(design)
    assert any("guard 'guardCanPublish' has no named-unit contract row" in d
               for d in g.drift)


# ----------------------- experiment G + Gx hardening -----------------------

def test_unenforced_invariant_is_error(design):
    """Review experiment G, now with structured matching."""
    _edit(os.path.join(design, "machines", "Widget.matrix.md"),
          "inv `widget-owned`", "inv `widget-possessed`")
    _edit(os.path.join(design, "machines", "Widget.matrix.md"),
          "surfaces `widget-owned`", "surfaces it")
    _edit(os.path.join(design, "BUILD.md"), "| widget-owned |", "| nothing |")
    g = mc.check_traceability(design)
    assert any("widget-owned" in e and "enforced by nothing" in e for e in g.errs)


def test_invariant_match_is_whole_token(design):
    """`widget-owned` must not be satisfied by `widget-owned-by-nobody`."""
    for old, new in (("inv `widget-owned`", "inv `widget-owned-by-nobody`"),
                     ("surfaces `widget-owned`", "surfaces nothing")):
        _edit(os.path.join(design, "machines", "Widget.matrix.md"), old, new)
    _edit(os.path.join(design, "BUILD.md"), "| widget-owned |", "| widget-owned-by-nobody |")
    g = mc.check_traceability(design)
    assert any("'widget-owned' is referenced by no" in e for e in g.errs)


def test_orphan_mapsto_reference_is_drift(design):
    _edit(os.path.join(design, "machines", "Widget.matrix.md"),
          "inv `widget-owned`", "inv `widget-owned` and `stale-invariant-ref`")
    g = mc.check_traceability(design)
    assert any("stale-invariant-ref" in d for d in g.drift)


def test_machine_state_not_in_enum_is_error(design):
    """The SagaStatus/FailedDirty class of drift: a TitleCase machine state
    that is not an enum value must fail, not pass as an advisory."""
    mp = os.path.join(design, "machines", "Widget.machine.json")
    m = json.load(open(mp, encoding="utf-8"))
    m["states"]["Archived"] = {"type": "final"}
    m["states"]["Draft"]["on"]["archive"] = {"target": "Archived"}
    json.dump(m, open(mp, "w", encoding="utf-8"), indent=1)
    g = mc.check_traceability(design)
    assert any("'Archived' is not a value of enum WidgetStatus" in e for e in g.errs)


def test_enum_value_without_state_is_error(design):
    _edit(os.path.join(design, "widget.modelith.yaml"),
          "      - name: Published", "      - name: Published\n      - name: Retired")
    g = mc.check_traceability(design)
    assert any("'Retired' has no machine state" in e for e in g.errs)


def test_machine_event_not_an_action_is_error(design):
    mp = os.path.join(design, "machines", "Widget.machine.json")
    m = json.load(open(mp, encoding="utf-8"))
    m["states"]["Draft"]["on"]["mysteryEvent"] = {"target": "Published"}
    json.dump(m, open(mp, "w", encoding="utf-8"), indent=1)
    g = mc.check_traceability(design)
    assert any("'mysteryEvent' is not a Modelith action" in e for e in g.errs)


def test_unmapped_machine_is_error(design):
    """A machine that matches no entity and declares no role cannot silently
    skip tracing (the hardcoded-lifecycle hole)."""
    src = os.path.join(design, "machines", "Widget.machine.json")
    dst = os.path.join(design, "machines", "Gadget.machine.json")
    shutil.copy(src, dst)
    import oracle_gen
    open(dst.replace(".machine.json", ".oracle.md"), "w",
         encoding="utf-8").write(oracle_gen.generate(dst))
    g = mc.check_traceability(design)
    assert any("Gadget.machine.json: maps to no Modelith entity" in e for e in g.errs)


def test_operational_role_is_accepted(design):
    mp = os.path.join(design, "machines", "Widget.machine.json")
    m = json.load(open(mp, encoding="utf-8"))
    m["_role"] = "operational"
    json.dump(m, open(mp, "w", encoding="utf-8"), indent=1)
    g = mc.check_traceability(design)
    assert not any("maps to no Modelith entity" in e for e in g.errs)
    # but the entity now has an enum and no lifecycle machine: that must fail
    assert any("has lifecycle enum WidgetStatus but no machine" in e for e in g.errs)


def test_placement_row_without_machine_is_error(design):
    _edit(os.path.join(design, "ARCHITECTURE.md"),
          "| `Widget` | in-process |", "| `Widget` | in-process |\n| `Gizmo` | actor |")
    g = mc.check_traceability(design)
    assert any("`Gizmo` has no machine" in e for e in g.errs)


def test_placement_waiver_is_accepted(design):
    _edit(os.path.join(design, "ARCHITECTURE.md"),
          "| `Widget` | in-process | db row | single writer |",
          "| `Widget` | in-process | db row | single writer |\n"
          "| `Gizmo` | pure function (no machine: stateless transform) | - | - |")
    g = mc.check_traceability(design)
    assert not any("Gizmo" in e for e in g.errs)


# ----------------------- experiment F1: import bypass -----------------------

def test_single_form_import_violation_is_caught(design, impl):
    """Review experiment F1: `import "widget/x"` (single form) was invisible."""
    p = os.path.join(impl, "internal", "app", "sneaky.go")
    with open(p, "w", encoding="utf-8") as f:
        f.write('package app\n\nimport "example.com/dbdriver"\n\n'
                'var _ = dbdriver.Name\n')
    g = mc.check_imports(design, impl)
    assert any("widget.app -> external.db is denied" in e for e in g.errs)


def test_paren_form_import_violation_is_caught(design, impl):
    p = os.path.join(impl, "internal", "app", "sneaky.go")
    with open(p, "w", encoding="utf-8") as f:
        f.write('package app\n\nimport (\n\t"example.com/dbdriver"\n)\n\n'
                'var _ = dbdriver.Name\n')
    g = mc.check_imports(design, impl)
    assert any("widget.app -> external.db is denied" in e for e in g.errs)


def test_undeclared_cross_boundary_edge_is_error(design, impl):
    p = os.path.join(impl, "internal", "store", "back.go")
    with open(p, "w", encoding="utf-8") as f:
        f.write('package store\n\nimport "widget/internal/app"\n\nvar _ = app.Run\n')
    g = mc.check_imports(design, impl)
    assert any("undeclared cross-boundary edge widget.store -> widget.app" in e
               for e in g.errs)


def test_import_of_unexposed_internals_is_error(design, impl):
    os.makedirs(os.path.join(impl, "internal", "store", "inner"), exist_ok=True)
    with open(os.path.join(impl, "internal", "store", "inner", "inner.go"),
              "w", encoding="utf-8") as f:
        f.write("package inner\n\nvar Name = \"x\"\n")
    with open(os.path.join(impl, "internal", "app", "deep.go"), "w", encoding="utf-8") as f:
        f.write('package app\n\nimport "widget/internal/store/inner"\n\nvar _ = inner.Name\n')
    g = mc.check_imports(design, impl)
    assert any("not in the exposes list of widget.store" in e for e in g.errs)


def test_source_outside_contract_is_error(design, impl):
    os.makedirs(os.path.join(impl, "internal", "rogue"), exist_ok=True)
    with open(os.path.join(impl, "internal", "rogue", "r.go"), "w", encoding="utf-8") as f:
        f.write("package rogue\n")
    g = mc.check_imports(design, impl)
    assert any("internal/rogue/r.go maps to no contract boundary" in e for e in g.errs)


def test_contract_ignore_globs_are_respected(design, impl):
    os.makedirs(os.path.join(impl, "internal", "scaffold"), exist_ok=True)
    with open(os.path.join(impl, "internal", "scaffold", "s.go"), "w", encoding="utf-8") as f:
        f.write("package scaffold\n")
    g = mc.check_imports(design, impl)
    assert not any("scaffold" in e for e in g.errs)
    assert g.counts.get("files ignored by contract") == 1


def test_python_imports_are_checked(design, tmp_path):
    impl = tmp_path / "pyimpl"
    (impl / "internal" / "app").mkdir(parents=True)
    (impl / "internal" / "store").mkdir(parents=True)
    (impl / "internal" / "app" / "app.py").write_text(
        "import example.com  # not internal\nfrom internal.store import store\n",
        encoding="utf-8")
    (impl / "internal" / "store" / "store.py").write_text("X = 1\n", encoding="utf-8")
    g = mc.check_imports(design, str(impl))
    # internal.store resolves inside widget.store; the edge app->store is allowed
    assert not any("denied" in e for e in g.errs), g.errs
    assert g.counts.get("python files checked") == 2
