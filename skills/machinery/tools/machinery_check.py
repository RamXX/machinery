#!/usr/bin/env python3
"""machinery check: deterministic verification gates for a machinery design.

No LLM. Pure static analysis over the design artifacts (and, with --impl, the
code). Gates:

  G2-c4       Architecture Contract valid, reconciled against workspace.dsl,
              every declared dependency covered by a mitigation row
  G3-machine  machines structurally sound (machine_lint), the committed oracle
              byte-identical to a fresh generation, the hand matrix structurally
              reconciled row by row, every guard/action/actor named in the
              named-unit contract table
  Gx-trace    machine states are exactly the Modelith lifecycle enum values,
              machine events are Modelith actions, every invariant is referenced
              by an enforcement artifact, every placement row has its machine
  G4-import   (needs --impl) code imports respect the Architecture Contract
              (Go, Python, TypeScript/JavaScript, Elixir, Rust extractors)

DESIGN RULE: absence is failure, not success. A gate that finds nothing to check
reports an ERROR, every gate prints what it actually verified, and a missing
artifact is a finding, never a silent skip. Run a subset with --gate while a
phase is still in flight; the default checks everything.

Known out of scope, by construction: coupling through shared database tables or
message-bus topics is invisible to import analysis; the event-contract table in
ARCHITECTURE.md is the artifact that governs those (see the c4 reference).

Exit non-zero on any ERROR or DRIFT. Warns and advisories do not fail the gate.

Usage: machinery_check.py <design-dir> [--impl <code-dir>] [--gate g2,g3,gx,g4]
"""
import argparse
import fnmatch
import glob
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import (walk_states, transitions_of, load_machine, lint_machine,  # noqa: E402
                          reconcile_matrix, namedunit_names, machine_unit_names,
                          parse_md_tables, find_col, IDENT)
import oracle_gen  # noqa: E402

try:
    import yaml
except ImportError:
    sys.exit("machinery_check: PyYAML is required (the contract and the domain model are YAML).\n"
             "Install it: `uv run --with pyyaml -- python ...` or `pip install pyyaml`.")


def _read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def _token_in(token, text):
    """Whole-token containment: `inv-1` must not match inside `inv-12`."""
    return re.search(rf"(?<![A-Za-z0-9_-]){re.escape(token)}(?![A-Za-z0-9_-])", text) is not None


class Gate:
    """Findings plus an explicit record of what was actually verified."""

    def __init__(self, title):
        self.title = title
        self.errs, self.drift, self.warns, self.notes = [], [], [], []
        self.counts = {}

    def count(self, label, n=1):
        self.counts[label] = self.counts.get(label, 0) + n

    def require_nonzero(self, label, what):
        if not self.counts.get(label):
            self.errs.append(f"nothing checked: {what}; an empty check is a failure, not a pass")

    def emit(self):
        print(f"== {self.title} ==")
        for e in self.errs:
            print(f"  ERROR  {e}")
        for d in self.drift:
            print(f"  DRIFT  {d}")
        for w in self.warns:
            print(f"  warn   {w}")
        for a in self.notes:
            print(f"  note   {a}")
        checked = ", ".join(f"{v} {k}" for k, v in self.counts.items() if v)
        print(f"  checked: {checked}" if checked else "  checked: nothing")
        if not (self.errs or self.drift or self.warns):
            print("  ok")
        return len(self.errs) + len(self.drift)


# ------------------------------ contract ----------------------------------

def load_contract(arch_path, g):
    """Locate and validate the Architecture Contract. Returns dict or None.

    The contract is the first ```yaml fence after a heading that contains
    'Architecture Contract' (fallback: any ```yaml fence whose first line is
    contract_version). A YAML block that merely happens to come first can no
    longer masquerade as the contract.
    """
    if not os.path.exists(arch_path):
        g.errs.append(f"{os.path.basename(arch_path)} does not exist")
        return None
    text = _read(arch_path)
    m = re.search(r"^#+[^\n]*architecture contract[^\n]*\n.*?```yaml\n(.*?)\n```",
                  text, re.S | re.I | re.M)
    if not m:
        m = re.search(r"```yaml\n(contract_version:.*?)\n```", text, re.S)
    if not m:
        g.errs.append("no Architecture Contract found (need a ```yaml fence under a heading "
                      "containing 'Architecture Contract', starting with contract_version)")
        return None
    try:
        c = yaml.safe_load(m.group(1))
    except yaml.YAMLError as e:
        g.errs.append(f"Architecture Contract is not valid YAML: {e}")
        return None
    if not isinstance(c, dict) or "contract_version" not in c:
        g.errs.append("Architecture Contract has no contract_version")
        return None
    if not c.get("boundaries"):
        g.errs.append("Architecture Contract declares no boundaries")
        return None
    return c


def contract_edges(rules, key, g=None):
    out = []
    for e in (rules.get(key) or []):
        mm = re.match(r'"?\s*([^\s"]+)\s*->\s*([^\s"#]+)', str(e))
        if mm:
            out.append((mm.group(1), mm.group(2)))
        elif g is not None:
            g.errs.append(f"unparseable {key} rule: {e!r} (expected 'src -> dst')")
    return out


def dsl_elements(dsl_path):
    """Parse workspace.dsl element declarations: id -> {kind, tags}."""
    els = {}
    decl = re.compile(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"
                      r"(person|softwareSystem|container|component)\b(.*)$")
    for line in _read(dsl_path).splitlines():
        m = decl.match(line)
        if not m:
            continue
        name, kind, rest = m.group(1), m.group(2), m.group(3)
        args = re.findall(r'"([^"]*)"', rest)
        tag_idx = 3 if kind in ("container", "component") else 2
        tags = set()
        if len(args) > tag_idx:
            tags = {t.strip() for t in args[tag_idx].split(",")}
        els[name] = {"kind": kind, "tags": tags, "display": args[0] if args else name}
    return els


def check_c4(design):
    g = Gate("G2-c4  Architecture Contract")
    arch = os.path.join(design, "ARCHITECTURE.md")
    c = load_contract(arch, g)
    if not c:
        return g
    boundaries = c.get("boundaries", [])
    externals = c.get("externals", []) or []

    ids = []
    for b in boundaries:
        if not isinstance(b, dict) or not b.get("id"):
            g.errs.append(f"boundary without an id: {b!r}")
            continue
        ids.append(b["id"])
        g.count("boundaries")
        if not b.get("code"):
            g.errs.append(f"boundary {b['id']!r} declares no code globs; G4 cannot map it")
    for bid in set(ids):
        if ids.count(bid) > 1:
            g.errs.append(f"duplicate boundary id {bid!r}")
    ext_ids = []
    for x in externals:
        if not isinstance(x, dict) or not x.get("id"):
            g.errs.append(f"externals entry without an id: {x!r}")
            continue
        ext_ids.append(x["id"])
        g.count("externals")
    declared = set(ids) | set(ext_ids)

    rules = c.get("dependency_rules", {}) or {}
    allow = contract_edges(rules, "allow", g)
    deny = contract_edges(rules, "deny", g)
    g.count("allow rules", len(allow))
    g.count("deny rules", len(deny))
    for src, dst in allow + deny:
        for side in (src, dst):
            if "*" in side:
                continue
            if side not in declared:
                hint = " (declare it under externals:)" if side.startswith("external") else ""
                g.errs.append(f"rule references undeclared boundary {side!r}{hint}")
    for e in set(allow) & set(deny):
        g.errs.append(f"edge {e[0]} -> {e[1]} is both allowed and denied")

    # workspace.dsl reconciliation
    dsl_path = os.path.join(design, "workspace.dsl")
    if not os.path.exists(dsl_path):
        g.errs.append("workspace.dsl does not exist; the contract has no model to bind to")
        return g
    els = dsl_elements(dsl_path)
    g.count("dsl elements", len(els))
    if not els:
        g.errs.append("workspace.dsl parsed but no elements found")

    def element_of(entry, default_from_id):
        el = entry.get("element", default_from_id)
        return el

    for b in boundaries:
        if not isinstance(b, dict) or not b.get("id"):
            continue
        el = element_of(b, b["id"].split(".")[-1])
        if el not in els:
            g.errs.append(f"boundary {b['id']!r} maps to no workspace.dsl element "
                          f"(looked for {el!r}; set element: explicitly if the id differs)")
        else:
            g.count("boundaries bound to dsl")
    for x in externals:
        if not isinstance(x, dict) or not x.get("id"):
            continue
        el = x.get("element")
        if el and el not in els:
            g.errs.append(f"external {x['id']!r} maps to element {el!r} not in workspace.dsl")

    # mitigation coverage: every infrastructure dependency has a mitigation row.
    # Required set: contract externals + DSL elements tagged Database/Queue/External.
    required = {x["id"]: x.get("element") for x in externals if isinstance(x, dict) and x.get("id")}
    infra_tags = {"Database", "Queue", "External"}
    infra = {name for name, e in els.items() if e["tags"] & infra_tags}
    text = _read(arch)
    mit_rows = []
    for header, rows in parse_md_tables(text):
        hl = " ".join(h.lower() for h in header)
        if "failure" in hl and "mitigation" in hl:
            mit_rows = rows
            break
    covered = set()
    for r in mit_rows:
        if not r:
            continue
        g.count("mitigation rows")
        for tok in re.findall(rf"`({IDENT}(?:\.{IDENT})*)`", r[0]):
            if tok in els or tok in required:
                covered.add(tok)
            else:
                g.errs.append(f"mitigation row names `{tok}`, which is neither a workspace.dsl "
                              f"element nor a declared external")
    need = set(required) | infra
    if need and not mit_rows:
        g.errs.append("no mitigation table found (header needs 'failure' and 'mitigation' "
                      "columns) although the design declares infrastructure dependencies")
    for dep in sorted(need):
        alt = required.get(dep)  # an external may be covered via its dsl element
        if dep in covered or (alt and alt in covered):
            g.count("dependencies with mitigation rows")
        else:
            g.errs.append(f"infrastructure dependency `{dep}` has no mitigation row "
                          f"(name it in the first column, backticked)")

    g.require_nonzero("boundaries", "no boundaries parsed")
    g.require_nonzero("dsl elements", "no workspace.dsl elements parsed")
    return g


# ------------------------------ machines ----------------------------------

def check_machines(design):
    g = Gate("G3-machine  machines + oracle")
    mdir = os.path.join(design, "machines")
    files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    if not files:
        g.errs.append(f"no *.machine.json under {mdir}")
        return g
    for path in files:
        base = os.path.basename(path)
        m, err = load_machine(path)
        if err:
            g.errs.append(err)
            continue
        g.count("machines")
        errs, warns, notes, counts = lint_machine(m, base)
        g.errs += errs
        g.warns += warns
        g.notes += notes
        g.count("transitions", counts["transitions"])

        # _exhaustive states carry a liveness claim TLC cannot verify (guards are
        # erased in the generated TLA+ model). Surface it prominently at the gate:
        # a false claim yields a green liveness proof for a machine that can deadlock.
        exhaustive = [n for p, n, node in walk_states(m.get("states"))
                      if isinstance(node, dict) and isinstance(node.get("_exhaustive"), str)
                      and node["_exhaustive"].strip()]
        if exhaustive:
            g.count("states relying on unproven _exhaustive liveness", len(exhaustive))
            g.warns.append(f"{base}: liveness for {', '.join(sorted(exhaustive))} rests on an "
                           f"UNPROVEN _exhaustive claim (guards are erased in the TLA+ model, so "
                           f"TLC cannot check it); verify the guard set is provably total, or add "
                           f"an unguarded fallback branch so the liveness proof becomes sound")

        # committed oracle must be byte-identical to a fresh generation
        opath = path.replace(".machine.json", ".oracle.md")
        fresh = oracle_gen.render(m, path)
        if not os.path.exists(opath):
            g.errs.append(f"{base}: no committed oracle ({os.path.basename(opath)}); "
                          f"run oracle_gen.py")
        elif _read(opath) != fresh:
            g.drift.append(f"{base}: committed oracle is stale (differs from a fresh "
                           f"generation); rerun oracle_gen.py")
        else:
            g.count("oracles fresh")

        # hand matrix: structural row-by-row reconciliation + named-unit coverage
        mpath = path.replace(".machine.json", ".matrix.md")
        if not os.path.exists(mpath):
            g.warns.append(f"{base}: no matrix file; named-unit contracts are unchecked "
                           f"(transitions are covered by the generated oracle)")
            continue
        mtext = _read(mpath)
        drift, nrows = reconcile_matrix(m, mtext, base)
        g.drift += drift
        g.count("matrix rows reconciled", nrows)
        declared = namedunit_names(mtext)
        guards, actions, actors = machine_unit_names(m)
        for kind, names in (("guard", guards), ("action", actions), ("actor", actors)):
            for name in sorted(names):
                if name in declared:
                    g.count("named units covered")
                else:
                    g.drift.append(f"{base}: {kind} {name!r} has no named-unit contract row "
                                   f"in {os.path.basename(mpath)}")
    g.require_nonzero("machines", "no machines parsed")
    g.require_nonzero("transitions", "no transitions parsed")
    return g


# ---------------------------- traceability --------------------------------

def load_modelith(design, g):
    paths = glob.glob(os.path.join(design, "*.modelith.yaml"))
    if not paths:
        g.errs.append("no *.modelith.yaml in the design directory")
        return None
    if len(paths) > 1:
        g.errs.append(f"multiple modelith models: {', '.join(sorted(os.path.basename(p) for p in paths))}")
        return None
    try:
        return yaml.safe_load(_read(paths[0]))
    except yaml.YAMLError as e:
        g.errs.append(f"{os.path.basename(paths[0])}: invalid YAML: {e}")
        return None


def check_traceability(design):
    g = Gate("Gx-trace  cross-layer traceability")
    dm = load_modelith(design, g)
    if not dm:
        return g
    enums = {k: [v["name"] for v in (val.get("values") or [])]
             for k, val in (dm.get("enums") or {}).items()}
    entities = dm.get("entities") or {}
    g.count("entities", len(entities))

    inv_ids = {i["id"] for i in (dm.get("invariants") or [])}
    actions_by_entity, enum_by_entity = {}, {}
    all_actions = set()
    for ename, e in entities.items():
        acts = {(a["name"] if isinstance(a, dict) else a) for a in (e.get("actions") or [])}
        actions_by_entity[ename] = acts
        all_actions |= acts
        for i in (e.get("invariants") or []):
            inv_ids.add(i["id"])
        for a in (e.get("attributes") or []):
            # the lifecycle enum is the enum-typed attribute named status/stage/state;
            # other enum-typed attributes (kind, role, type) are categorical, not
            # lifecycles, and demand no machine
            if isinstance(a, dict) and a.get("type") in enums \
                    and a.get("name") in ("status", "stage", "state"):
                enum_by_entity[ename] = a["type"]

    mdir = os.path.join(design, "machines")
    machine_files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    if not machine_files:
        g.errs.append(f"no machines under {mdir} to trace")
        return g

    machine_names = set()
    claimed = set()  # entities actually claimed by a lifecycle machine
    for path in machine_files:
        base = os.path.basename(path)
        name = base.replace(".machine.json", "")
        machine_names.add(name)
        m, err = load_machine(path)
        if err:
            g.errs.append(err)
            continue
        entity = m.get("_lifecycle_of") or (name if name in entities else None)
        role = m.get("_role")
        if role == "operational":
            entity = None
        if entity is None and role != "operational":
            g.errs.append(f"{base}: maps to no Modelith entity and is not declared "
                          f"operational (set _lifecycle_of: <Entity> or _role: operational)")
            continue
        if role == "operational":
            g.count("operational machines")
            continue
        if entity not in entities:
            g.errs.append(f"{base}: _lifecycle_of {entity!r} is not a Modelith entity")
            continue
        claimed.add(entity)
        enum_name = enum_by_entity.get(entity)
        if not enum_name:
            g.errs.append(f"{base}: entity {entity!r} has no enum-typed lifecycle attribute")
            continue
        vals = set(enums.get(enum_name) or [])
        top = [n for p, n, _ in walk_states(m.get("states")) if "." not in p]
        domain_states = {n for n in top if n[:1].isupper()}
        overlay = [n for n in top if not n[:1].isupper()]
        for s in sorted(domain_states - vals):
            g.errs.append(f"{base}: domain state {s!r} is not a value of enum {enum_name} "
                          f"(overlay states are lowerCamel by convention)")
        for v in sorted(vals - domain_states):
            g.errs.append(f"{base}: enum {enum_name} value {v!r} has no machine state; "
                          f"the lifecycle is incomplete")
        if overlay:
            g.notes.append(f"{name}: {len(overlay)} operational-overlay states "
                           f"({', '.join(overlay)})")
        events = set()
        for _, _, node in walk_states(m.get("states")):
            events |= set((node.get("on") or {}).keys())
        for ev in sorted(events):
            if ev not in actions_by_entity.get(entity, set()):
                g.errs.append(f"{base}: event {ev!r} is not a Modelith action of {entity}")
        g.count("lifecycle machines traced")

    # every entity with a lifecycle enum has a machine that claims it
    for ename, enum_name in sorted(enum_by_entity.items()):
        if ename not in claimed:
            g.errs.append(f"entity {ename} has lifecycle enum {enum_name} but no machine "
                          f"({ename}.machine.json); model the lifecycle or drop the enum")
        else:
            g.count("lifecycle entities with machines")

    # placement table: every stateful component row names a machine
    arch = os.path.join(design, "ARCHITECTURE.md")
    if os.path.exists(arch):
        for header, rows in parse_md_tables(_read(arch)):
            hl = " ".join(h.lower() for h in header)
            if "placement" in hl and "persistence" in hl:
                for r in rows:
                    if not r:
                        continue
                    # the FIRST backticked token is the component; later backticked
                    # words in the same cell (e.g. an attribute named in the waiver
                    # reason) are prose, not extra components
                    named = re.findall(rf"`({IDENT})`", r[0])[:1]
                    if not named:
                        g.errs.append(f"placement row names no component in backticks: "
                                      f"{r[0]!r}")
                    for comp in named:
                        if comp in machine_names:
                            g.count("placement rows with machines")
                        elif "(no machine:" in " ".join(r):
                            g.count("placement rows waived")
                        else:
                            g.errs.append(f"placement row component `{comp}` has no machine "
                                          f"and no '(no machine: <reason>)' waiver")
                break

    # invariants: enforced somewhere structured (matrix or BUILD.md table cells).
    # Distinguish UNIT-BACKED enforcement (the invariant is cited against a concrete
    # guard/action/actor in a matrix named-unit "maps to" column, so it has an
    # implementation unit with a contract and a test) from ATTESTED enforcement (it
    # appears only in a prose/BUILD table cell, e.g. "enforced by (structural)").
    # Both satisfy traceability, but the split is reported so a green "enforced"
    # count cannot silently mean "an id was typed into a cell".
    cells, unit_cells = [], []
    for f in glob.glob(os.path.join(mdir, "*.matrix.md")):
        for header, rows in parse_md_tables(_read(f)):
            mi = find_col(header, "maps to")
            for r in rows:
                cells += r
                if mi is not None and mi < len(r):
                    unit_cells.append(r[mi])
    build = os.path.join(design, "BUILD.md")
    if os.path.exists(build):
        for header, rows in parse_md_tables(_read(build)):
            for r in rows:
                cells += r
    else:
        g.warns.append("BUILD.md not present; invariant enforcement is checked against "
                       "the matrices only (fine before Phase 4)")
    corpus = "\n".join(cells)
    unit_corpus = "\n".join(unit_cells)
    if not inv_ids:
        g.errs.append("the domain model declares no invariants; nothing constrains the design")
    for iid in sorted(inv_ids):
        if _token_in(iid, corpus):
            g.count("invariants enforced")
            if _token_in(iid, unit_corpus):
                g.count("invariants unit-backed (guard/action/actor)")
            else:
                g.count("invariants attested (structural/prose)")
        else:
            g.errs.append(f"invariant {iid!r} is referenced by no matrix or BUILD.md table; "
                          f"it is enforced by nothing")

    # orphan references: backticked kebab tokens in maps-to columns must exist
    known = set(inv_ids) | all_actions | set(entities)
    for vs in enums.values():
        known |= set(vs)
    for f in glob.glob(os.path.join(mdir, "*.matrix.md")):
        for header, rows in parse_md_tables(_read(f)):
            mi = find_col(header, "maps to")
            if mi is None:
                continue
            for r in rows:
                if mi >= len(r):
                    continue
                for tok in re.findall(r"`([a-z][a-z0-9]*(?:-[a-z0-9]+)+)`", r[mi]):
                    if tok not in known:
                        g.drift.append(f"{os.path.basename(f)}: maps-to references "
                                       f"`{tok}`, which is not a declared invariant "
                                       f"(typo or a stale reference)")
    g.require_nonzero("invariants enforced", "no invariant was traced to an enforcement artifact")
    return g


# ------------------------------ imports -----------------------------------

TEST_FILE_PATTERNS = ("*_test.go", "*_test.py", "test_*.py", "*.test.ts", "*.test.tsx",
                      "*.test.js", "*_test.exs", "*_spec.rb")


def _is_test_file(rel):
    base = os.path.basename(rel)
    return any(fnmatch.fnmatch(base, p) for p in TEST_FILE_PATTERNS)


def _match_glob(rel, pattern):
    pattern = pattern.rstrip("/")
    if fnmatch.fnmatch(rel, pattern):
        return True
    static = pattern.replace("/**", "").replace("/*", "").rstrip("/")
    return rel == static or rel.startswith(static + "/")


def boundary_of(rel, pkgmap):
    best = None
    for pattern, bid in pkgmap:
        if _match_glob(rel, pattern):
            static = pattern.replace("/**", "").replace("/*", "")
            if best is None or len(static) > best[0]:
                best = (len(static), bid)
    return best[1] if best else None


def go_module_name(impl):
    gomod = os.path.join(impl, "go.mod")
    if not os.path.exists(gomod):
        return None
    m = re.search(r"^module\s+(\S+)", _read(gomod), re.M)
    return m.group(1) if m else None


def go_imports(text):
    """All import paths in a Go file: single-form, aliased, and block imports."""
    out = []
    for blk in re.findall(r"^import\s*\((.*?)\)", text, re.S | re.M):
        for line in blk.splitlines():
            m = re.search(r'(?:^|\s)(?:[\w.]+\s+)?"([^"]+)"', line)
            if m:
                out.append(m.group(1))
    for m in re.finditer(r'^import\s+(?:[\w.]+\s+)?"([^"]+)"', text, re.M):
        out.append(m.group(1))
    return out


def py_imports(text, rel):
    out = []
    for m in re.finditer(r"^\s*import\s+([\w.]+)", text, re.M):
        out.append(m.group(1).replace(".", "/"))
    for m in re.finditer(r"^\s*from\s+([\w.]+)\s+import\b", text, re.M):
        mod = m.group(1)
        if mod.startswith("."):
            base = os.path.dirname(rel)
            for _ in range(len(mod) - len(mod.lstrip(".")) - 1):
                base = os.path.dirname(base)
            mod = os.path.join(base, mod.lstrip(".").replace(".", "/")).strip("/")
            out.append(mod)
        else:
            out.append(mod.replace(".", "/"))
    return out


def ts_imports(text, rel):
    out = []
    for m in re.finditer(r"""(?:from|import|require\()\s*['"]([^'"]+)['"]""", text):
        spec = m.group(1)
        if spec.startswith("."):
            out.append(os.path.normpath(os.path.join(os.path.dirname(rel), spec)))
        else:
            out.append(spec)
    return out


def ex_modules(text):
    out = []
    for m in re.finditer(r"^\s*(?:alias|import|use|require)\s+([A-Z][\w.]*)", text, re.M):
        out.append(m.group(1))
    return out


def rust_imports(text, rel):
    out = []
    for m in re.finditer(r"^\s*use\s+([\w:]+)", text, re.M):
        path = m.group(1)
        if path.startswith("crate::"):
            out.append("src/" + path[len("crate::"):].replace("::", "/"))
        else:
            out.append(path.split("::")[0])
    return out


LANG_EXTS = {".go": "go", ".py": "python", ".ts": "ts", ".tsx": "ts", ".js": "ts",
             ".jsx": "ts", ".ex": "elixir", ".exs": "elixir", ".rs": "rust"}


def check_imports(design, impl):
    g = Gate("G4-import  code respects the contract")
    if not os.path.isdir(impl):
        g.errs.append(f"--impl {impl!r} is not a directory")
        return g
    cg = Gate("_")
    c = load_contract(os.path.join(design, "ARCHITECTURE.md"), cg)
    if not c:
        g.errs += cg.errs or ["no contract to check against"]
        return g
    boundaries = [b for b in c.get("boundaries", []) if isinstance(b, dict) and b.get("id")]
    externals = [x for x in (c.get("externals") or []) if isinstance(x, dict) and x.get("id")]
    ignore = c.get("ignore") or []
    pkgmap = [(code, b["id"]) for b in boundaries for code in (b.get("code") or [])]
    exposes = {b["id"]: b.get("exposes") for b in boundaries}
    ext_by_prefix = [(p, x["id"]) for x in externals for p in (x.get("imports") or [])]
    ext_modules = [(mprefix, x["id"]) for x in externals for mprefix in (x.get("modules") or [])]
    bound_modules = [(mprefix, b["id"]) for b in boundaries for mprefix in (b.get("modules") or [])]

    rules = c.get("dependency_rules", {}) or {}
    allow = contract_edges(rules, "allow")
    deny = contract_edges(rules, "deny")

    def match_rule(rules_, src, dst):
        return any(fnmatch.fnmatch(src, rs) and fnmatch.fnmatch(dst, rd) for rs, rd in rules_)

    go_module = go_module_name(impl)

    def internal_target(ref):
        """Map an import reference to (boundary, normalized-path) or (None, None)."""
        if go_module and (ref == go_module or ref.startswith(go_module + "/")):
            rel = ref[len(go_module):].lstrip("/")
            return boundary_of(rel, pkgmap), rel
        for prefix, bid in bound_modules:
            if ref == prefix or ref.startswith(prefix + "."):
                return bid, ref
        b = boundary_of(ref, pkgmap)
        if b:
            return b, ref
        for ext in ("", ".py", ".ts", ".tsx", ".js", ".rs"):
            b = boundary_of(ref + ext, pkgmap)
            if b:
                return b, ref + ext
        return None, None

    def external_target(ref):
        for prefix, xid in ext_by_prefix:
            if ref == prefix or ref.startswith(prefix.rstrip("/") + "/"):
                return xid
        for prefix, xid in ext_modules:
            if ref == prefix or ref.startswith(prefix + "."):
                return xid
        return None

    edge_hits = {}
    files = [p for p in glob.glob(os.path.join(impl, "**", "*"), recursive=True)
             if os.path.isfile(p) and os.path.splitext(p)[1] in LANG_EXTS]
    for path in sorted(files):
        rel = os.path.relpath(path, impl)
        if any(_match_glob(rel, ig) for ig in ignore):
            g.count("files ignored by contract")
            continue
        if _is_test_file(rel):
            g.count("test files skipped")
            continue
        lang = LANG_EXTS[os.path.splitext(path)[1]]
        src_b = boundary_of(rel, pkgmap)
        text = _read(path)
        if src_b is None:
            if lang == "elixir":
                # an elixir file may belong to a boundary via its module name
                mods = re.findall(r"^\s*defmodule\s+([A-Z][\w.]*)", text, re.M)
                for mod in mods:
                    for prefix, bid in bound_modules:
                        if mod == prefix or mod.startswith(prefix + "."):
                            src_b = bid
                            break
        if src_b is None:
            g.errs.append(f"source file {rel} maps to no contract boundary; add it to a "
                          f"boundary's code globs or to the contract ignore list")
            continue
        g.count(f"{lang} files checked")

        refs = {"go": lambda: go_imports(text),
                "python": lambda: py_imports(text, rel),
                "ts": lambda: ts_imports(text, rel),
                "elixir": lambda: ex_modules(text),
                "rust": lambda: rust_imports(text, rel)}[lang]()
        for ref in refs:
            dst_b, norm = internal_target(ref)
            if dst_b is None:
                dst_b = external_target(ref)
                norm = ref
                if dst_b is None:
                    if go_module and ref.startswith(go_module + "/"):
                        g.errs.append(f"{rel}: imports {ref}, which maps to no contract "
                                      f"boundary (code outside the contract)")
                    continue  # stdlib or an undeclared third-party lib: not a boundary edge
            g.count("imports resolved")
            if dst_b == src_b:
                continue
            # exposes: importing a boundary's non-exposed internals is a violation.
            # A file entry (pkg/api.go) exposes exactly its package directory; a
            # glob entry matches the normalized import (extension-tolerant).
            exp = exposes.get(dst_b)
            if exp and norm is not None:
                exposed_dirs = {os.path.dirname(e) for e in exp if "*" not in e}
                ok = norm in exposed_dirs or any(
                    fnmatch.fnmatch(cand, e)
                    for e in exp
                    for cand in (norm, norm + ".py", norm + ".ts", norm + ".js", norm + ".rs"))
                if not ok:
                    g.errs.append(f"{rel}: imports {ref}, which is not in the exposes list "
                                  f"of {dst_b}")
            edge = (src_b, dst_b)
            if edge in edge_hits:
                continue
            edge_hits[edge] = rel
            denied = match_rule(deny, src_b, dst_b)
            allowed = match_rule(allow, src_b, dst_b)
            if denied and not allowed:
                g.errs.append(f"{src_b} -> {dst_b} is denied by the contract (seen in {rel}); "
                              f"either the code violates the boundary or the contract needs "
                              f"an explicit allow")
            elif not allowed and not denied:
                g.errs.append(f"undeclared cross-boundary edge {src_b} -> {dst_b} (seen in "
                              f"{rel}); add an explicit allow or deny to the contract")
            else:
                g.count("edges verified")

    if not any(k.endswith("files checked") for k, v in g.counts.items() if v):
        g.errs.append(f"no source files under {impl} mapped to any contract boundary; "
                      f"the gate checked nothing")
    g.require_nonzero("imports resolved", "no imports were resolved against the contract")
    return g


# ------------------------------- main --------------------------------------

def main():
    ap = argparse.ArgumentParser(description="deterministic machinery gates")
    ap.add_argument("design", help="design directory (with ARCHITECTURE.md, machines/, ...)")
    ap.add_argument("--impl", help="implementation directory for G4-import")
    ap.add_argument("--gate", default=None,
                    help="comma list of gates to run: g2,g3,gx,g4 (default: all applicable)")
    args = ap.parse_args()
    if not os.path.isdir(args.design):
        sys.exit(f"machinery_check: design directory {args.design!r} does not exist")
    gates = set((args.gate or "g2,g3,gx,g4").lower().split(","))
    unknown = gates - {"g2", "g3", "gx", "g4"}
    if unknown:
        sys.exit(f"machinery_check: unknown gate(s): {', '.join(sorted(unknown))}")
    if "g4" in gates and (args.gate or "").find("g4") >= 0 and not args.impl:
        sys.exit("machinery_check: --gate g4 requires --impl")

    fail = 0
    if "g2" in gates:
        fail += check_c4(args.design).emit()
    if "g3" in gates:
        fail += check_machines(args.design).emit()
    if "gx" in gates:
        fail += check_traceability(args.design).emit()
    if "g4" in gates and args.impl:
        fail += check_imports(args.design, args.impl).emit()

    print(f"\n{fail} blocking (ERROR/DRIFT) finding(s)")
    sys.exit(1 if fail else 0)


if __name__ == "__main__":
    main()
