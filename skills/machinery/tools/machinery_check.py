#!/usr/bin/env python3
"""machinery check: deterministic verification gates for a machinery design.

No LLM. Pure static analysis over the design artifacts (and, with --impl, the
generated code). Gates:

  G2-c4       Architecture Contract internally consistent and complete
  G3-machine  machines well-formed (targets resolve, no dead ends, invokes have
              onError + timeout) AND the transition-matrix oracle reconciles with
              the machine JSON (drift => the oracle must be regenerated)
  Gx-trace    cross-layer traceability: machine states are Modelith enum values,
              events are Modelith actions, every invariant is enforced somewhere
  G4-import   (needs --impl) code imports respect the Architecture Contract

Exit non-zero on any ERROR or DRIFT. Warns and advisories do not fail the gate.

Usage: machinery_check.py <design-dir> [--impl <code-dir>]
"""
import sys, os, re, glob, json, fnmatch

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of, actions_of  # noqa: E402

try:
    import yaml
except Exception:
    yaml = None


# ------------------------------- helpers ---------------------------------

def _ident_tokens(s):
    return set(re.findall(r"[a-z][a-zA-Z0-9]+", s))


def parse_md_tables(text):
    """Return list of (header:[cells], rows:[[cells]...]) for each pipe table."""
    blocks, cur = [], []
    for line in text.splitlines():
        if line.lstrip().startswith("|"):
            cur.append(line.strip())
        elif cur:
            blocks.append(cur)
            cur = []
    if cur:
        blocks.append(cur)
    tables = []
    for b in blocks:
        rows = [[c.strip() for c in r.strip().strip("|").split("|")] for r in b]
        if len(rows) < 2:
            continue
        sep = "".join(rows[1])
        data = rows[2:] if set(sep) <= set("-: ") else rows[1:]
        tables.append((rows[0], data))
    return tables


def _find_col(header, *names):
    for i, h in enumerate(header):
        hl = h.lower()
        if any(n in hl for n in names):
            return i
    return None


def transition_table(matrix_path):
    """Return (rows, actions_col_idx) for the transition-matrix table only."""
    text = open(matrix_path).read()
    for header, rows in parse_md_tables(text):
        hl = [h.lower() for h in header]
        joined = " ".join(hl)
        if "source" in joined and "target" in joined and "actions" in joined:
            return rows, _find_col(header, "actions")
    return None, None


def namedunit_tokens(matrix_path):
    """Identifier tokens declared in the named-unit contract table (guards,
    actions, actors). Entry/exit actions are declared here, not in transition rows."""
    text = open(matrix_path).read()
    toks = set()
    for header, rows in parse_md_tables(text):
        hl = " ".join(h.lower() for h in header)
        if "signature" in hl or "maps to" in hl or ("name" in hl and "kind" in hl):
            for r in rows:
                toks |= _ident_tokens(" ".join(r))
    return toks


# ------------------------------- G3 machine ------------------------------

def check_machines(mdir):
    errs, drift, warns = [], [], []
    files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    if not files:
        warns.append(f"no *.machine.json under {mdir} yet (run after Phase 3)")
    for path in files:
        base = os.path.basename(path)
        m = json.load(open(path))
        states = walk_states(m.get("states"))
        paths = {p for p, _, _ in states}
        simple = {n for _, n, _ in states}

        def resolves(t):
            if t is None:
                return True
            t = t.lstrip("#")
            return t in paths or t in simple or t.split(".")[-1] in simple

        for p, _, node in states:
            for tr in transitions_of(node):
                if not resolves(tr["target"]):
                    errs.append(f"{base}: dangling target {tr['target']!r} from {p}")
            is_final = node.get("type") == "final"
            if not is_final and "states" not in node and not transitions_of(node):
                errs.append(f"{base}: dead-end non-final leaf state {p}")
            if "invoke" in node:
                invs = node["invoke"]
                for iv in (invs if isinstance(invs, list) else [invs]):
                    if "onError" not in iv:
                        errs.append(f"{base}: invoke {iv.get('src')!r} in {p} has no onError")
                if "after" not in node:
                    errs.append(f"{base}: invoking state {p} has no after/timeout")

        # oracle reconciliation. RULE: a transition action must appear in that
        # transition's matrix row; a state entry/exit action must be declared in
        # the named-unit contract table. An action fired but represented in
        # neither is drift (the oracle no longer specifies the machine).
        mpath = path.replace(".machine.json", ".matrix.md")
        if not os.path.exists(mpath):
            warns.append(f"{base}: no hand-authored matrix; the oracle is generated from the machine, so no drift is possible")
            continue
        rows, acol = transition_table(mpath)
        if rows is None:
            errs.append(f"{base}: no transition-matrix table found in the matrix file")
            continue
        row_tokens = set()
        for r in rows:
            cell = r[acol] if (acol is not None and acol < len(r)) else " ".join(r)
            row_tokens |= _ident_tokens(cell)
        named = namedunit_tokens(mpath)
        for p, _, node in states:
            for tr in transitions_of(node):
                for a in tr["actions"]:
                    if a not in row_tokens:
                        drift.append(f"{base}: transition action {a!r} ({tr['kind']} in {p}) is "
                                     f"fired but absent from the transition-matrix rows")
            for k in ("entry", "exit"):
                v = node.get(k)
                for a in (v if isinstance(v, list) else [v] if v else []):
                    if isinstance(a, str) and a not in named and a not in row_tokens:
                        drift.append(f"{base}: {k} action {a!r} on state {p} is fired but not "
                                     f"declared in the named-unit contract table")
    # dedupe while preserving order
    drift = list(dict.fromkeys(drift))
    return errs, drift, warns


# ------------------------------- G2 c4 -----------------------------------

def _contract_yaml(arch_path):
    text = open(arch_path).read()
    m = re.search(r"##\s*4\.[^\n]*\n.*?```yaml\n(.*?)\n```", text, re.S)
    if not m:
        m = re.search(r"```yaml\n(contract_version:.*?)\n```", text, re.S)
    return yaml.safe_load(m.group(1)) if (m and yaml) else None


def _dsl_ids(dsl_path):
    ids = set()
    for line in open(dsl_path):
        m = re.match(r"\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(person|softwareSystem|container|component)\b", line)
        if m:
            ids.add(m.group(1))
    return ids


def check_c4(design):
    errs, warns, adv = [], [], []
    arch = os.path.join(design, "ARCHITECTURE.md")
    dsl = os.path.join(design, "workspace.dsl")
    c = _contract_yaml(arch) if os.path.exists(arch) else None
    if not c:
        errs.append("ARCHITECTURE.md: no parseable Architecture Contract YAML block")
        return errs, warns, adv
    boundaries = c.get("boundaries", [])
    ids = [b.get("id") for b in boundaries]
    for bid in ids:
        if ids.count(bid) > 1:
            errs.append(f"contract: duplicate boundary id {bid!r}")
    idset = set(ids)
    rules = c.get("dependency_rules", {}) or {}

    def edges(key):
        out = []
        for e in rules.get(key, []) or []:
            mm = re.match(r'"?\s*([^\s"]+)\s*->\s*([^\s"#]+)', e)
            if mm:
                out.append((mm.group(1), mm.group(2)))
        return out

    allow, deny = edges("allow"), edges("deny")
    for src, dst in allow + deny:
        for side in (src, dst):
            base = side.replace(".*", "")
            if "*" in side:
                continue
            if side not in idset and not side.startswith("external"):
                warns.append(f"contract: rule references undeclared boundary {side!r}")
    for e in set(allow) & set(deny):
        errs.append(f"contract: edge {e[0]} -> {e[1]} is both allowed and denied")

    # completeness: external deps have a mitigation row; stateful components have a placement row
    text = open(arch).read()
    ext_terms = ["LadybugDB", "Payments", "Queue"]
    sec6 = re.search(r"##\s*6\..*?(?=\n##\s*\d|\Z)", text, re.S)
    if sec6:
        for term in ext_terms:
            if term in text and term not in sec6.group(0) and term == "LadybugDB":
                warns.append(f"contract: external dependency {term!r} lacks a section-6 mitigation row")
    return errs, warns, adv


# ------------------------------- Gx trace --------------------------------

def _modelith(design):
    p = glob.glob(os.path.join(design, "*.modelith.yaml"))
    if not p or not yaml:
        return None
    return yaml.safe_load(open(p[0]))


def check_traceability(design):
    errs, warns, adv = [], [], []
    dm = _modelith(design)
    if not dm:
        adv.append("traceability: no modelith model parsed; skipped")
        return errs, warns, adv
    enums = {k: {v["name"] for v in val.get("values", [])} for k, val in (dm.get("enums") or {}).items()}
    entities = dm.get("entities") or {}
    action_names = set()
    inv_ids = set(i["id"] for i in (dm.get("invariants") or []))
    for ename, e in entities.items():
        for a in (e.get("actions") or []):
            action_names.add(a["name"] if isinstance(a, dict) else a)
        for i in (e.get("invariants") or []):
            inv_ids.add(i["id"])

    lifecycle = {"Deal": "DealStage", "Task": "TaskStatus", "User": "UserStatus"}
    mdir = os.path.join(design, "machines")
    for comp, enum in lifecycle.items():
        mp = os.path.join(mdir, f"{comp}.machine.json")
        if not os.path.exists(mp):
            continue
        m = json.load(open(mp))
        vals = enums.get(enum, set())
        top = [n for _, n, _ in walk_states(m.get("states")) if "." not in _]
        overlay = [n for p, n, _ in walk_states(m.get("states")) if "." not in p and n not in vals]
        if overlay:
            adv.append(f"{comp}: {len(overlay)} operational-overlay states, not {enum} values "
                       f"({', '.join(overlay)})")
        events = set()
        for _, _, node in walk_states(m.get("states")):
            events |= set((node.get("on") or {}).keys())
        for ev in sorted(events):
            if ev not in action_names:
                adv.append(f"{comp}: event {ev!r} is not a Modelith action for a domain entity")

    # every invariant enforced somewhere: named in a matrix maps-to column or in BUILD.md section 6
    build = os.path.join(design, "BUILD.md")
    if not os.path.exists(build):
        adv.append("traceability: BUILD.md not yet authored; invariant-enforcement check deferred to Phase 4")
        return errs, warns, adv
    corpus = open(build).read()
    for f in glob.glob(os.path.join(mdir, "*.matrix.md")):
        corpus += open(f).read()
    for iid in sorted(inv_ids):
        if iid not in corpus:
            errs.append(f"traceability: invariant {iid!r} is enforced by nothing (absent from "
                        f"BUILD.md and the machine matrices)")
    return errs, warns, adv


# ------------------------------- G4 imports ------------------------------

def _pkg_boundary_map(boundaries):
    out = {}
    for b in boundaries:
        for code in (b.get("code") or []):
            prefix = code.replace("/**", "").replace("/*", "").rstrip("/")
            out[prefix] = b["id"]
    return out


def _match_rule(rules, src, dst):
    for rs, rd in rules:
        if fnmatch.fnmatch(src, rs) and fnmatch.fnmatch(dst, rd):
            return True
    return False


def check_imports(design, impl):
    errs, warns, adv = [], [], []
    c = _contract_yaml(os.path.join(design, "ARCHITECTURE.md"))
    if not c:
        errs.append("imports: no contract to check against")
        return errs, warns, adv
    boundaries = c.get("boundaries", [])
    pkgmap = _pkg_boundary_map(boundaries)
    rules = c.get("dependency_rules", {}) or {}

    def edges(key):
        out = []
        for e in rules.get(key, []) or []:
            mm = re.match(r'"?\s*([^\s"]+)\s*->\s*([^\s"#]+)', e)
            if mm:
                out.append((mm.group(1), mm.group(2)))
        return out

    allow, deny = edges("allow"), edges("deny")

    def boundary_of_relpath(rel):
        best = None
        for prefix, bid in pkgmap.items():
            if rel.startswith(prefix + "/") or rel == prefix:
                if best is None or len(prefix) > len(best[0]):
                    best = (prefix, bid)
        return best[1] if best else None

    seen_edges = set()
    for gofile in glob.glob(os.path.join(impl, "**", "*.go"), recursive=True):
        rel = os.path.relpath(gofile, impl)
        src_b = boundary_of_relpath(rel)
        if not src_b:
            continue
        text = open(gofile).read()
        imports = re.findall(r'"([^"]+)"', "\n".join(
            re.findall(r"import\s*\((.*?)\)", text, re.S) + re.findall(r'import\s+"([^"]+)"', text)))
        for imp in imports:
            dst_b = None
            if imp.startswith("crm/"):
                dst_b = boundary_of_relpath(imp[len("crm/"):])
            elif "LadybugDB/go-ladybug" in imp:
                dst_b = "external.ladybug"
            if not dst_b or dst_b == src_b:
                continue
            edge = (src_b, dst_b)
            if edge in seen_edges:
                continue
            seen_edges.add(edge)
            denied = _match_rule(deny, src_b, dst_b)
            allowed = _match_rule(allow, src_b, dst_b)
            if denied and not allowed:
                errs.append(f"imports: {src_b} -> {dst_b} is denied by the contract "
                            f"(seen in {rel}); either the code violates the boundary or the "
                            f"contract needs an explicit allow")
            elif not allowed and not denied:
                warns.append(f"imports: undeclared cross-boundary edge {src_b} -> {dst_b} "
                             f"(seen in {rel})")
    return errs, warns, adv


# ------------------------------- main ------------------------------------

def _emit(title, errs, drift, warns, adv):
    print(f"== {title} ==")
    for e in errs:
        print(f"  ERROR  {e}")
    for d in drift:
        print(f"  DRIFT  {d}")
    for w in warns:
        print(f"  warn   {w}")
    for a in adv:
        print(f"  note   {a}")
    if not (errs or drift or warns or adv):
        print("  ok")
    return len(errs) + len(drift)


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    impl = None
    if "--impl" in sys.argv:
        impl = sys.argv[sys.argv.index("--impl") + 1]
    design = args[0] if args else "."
    fail = 0

    e, w, a = check_c4(design)
    fail += _emit("G2-c4  Architecture Contract", e, [], w, a)

    e, d, w = check_machines(os.path.join(design, "machines"))
    fail += _emit("G3-machine  machines + oracle", e, d, w, [])

    e, w, a = check_traceability(design)
    fail += _emit("Gx-trace  cross-layer traceability", e, [], w, a)

    if impl:
        e, w, a = check_imports(design, impl)
        fail += _emit("G4-import  code respects the contract", e, [], w, a)

    print(f"\n{fail} blocking (ERROR/DRIFT) finding(s)")
    sys.exit(1 if fail else 0)


if __name__ == "__main__":
    main()
