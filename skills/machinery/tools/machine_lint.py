#!/usr/bin/env python3
"""machine_lint: deterministic checks for machinery Phase-3 XState machines.

No LLM. This module is both a CLI and the shared IR for the other tools
(oracle_gen, tla_gen, machinery_check). Layers:

  1. Structural lint over the finite state graph: valid JSON, only the supported
     XState subset (unknown keys are ERRORS, not silently skipped), every
     transition target resolves unambiguously, every state is reachable, no
     dead-end non-final leaf, every invoke has onError and an after/timeout,
     no branch is shadowed by an earlier unguarded branch, and every fully
     guarded `always` list either has an unguarded escape or carries an explicit
     `_exhaustive` justification (the liveness side condition, made visible).
  2. JSON <-> matrix reconciliation, structural and bidirectional: each machine
     transition must appear as a matrix row with the same source, trigger,
     guard shape, target, and actions, and each matrix row must correspond to a
     machine transition. A bag-of-words match is not reconciliation.

Supported XState v5 subset (JSON-serializable, one machine per file):
  root:  id, initial, context, states, description, meta, version,
         _comment, _delays, _lifecycle_of, _role, _component
  state: on, after, always, invoke, entry, exit, states, initial, type,
         id, meta, description, tags, onDone, output, _comment, _exhaustive,
         _ignores ({event: reason} - the explicit-ignore notation for event
         completeness: every domain state must handle or explicitly ignore
         every event the machine reacts to)
  type:  atomic (default), compound (has states), final
  invoke: src, input, id, onDone, onError, _comment
Parallel and history states, root-level `on`, and non-string guards are NOT
supported and are reported as errors; do not silently narrow them.

Exit non-zero on any ERROR or DRIFT finding, so it can gate Phase 3.

Usage: machine_lint.py <machines-dir>
"""
import json, sys, os, re, glob

ROOT_KEYS = {"id", "initial", "context", "states", "description", "meta", "version",
             "_comment", "_delays", "_lifecycle_of", "_role", "_component"}
STATE_KEYS = {"on", "after", "always", "invoke", "entry", "exit", "states", "initial",
              "type", "id", "meta", "description", "tags", "onDone", "output",
              "_comment", "_exhaustive", "_ignores"}
INVOKE_KEYS = {"src", "input", "id", "onDone", "onError", "_comment"}
STATE_TYPES = {None, "atomic", "compound", "final"}

IDENT = r"[A-Za-z_][A-Za-z0-9_]*"


# ------------------------------- shared IR --------------------------------

def walk_states(states, prefix=""):
    """Yield (path, simple_name, node) for every state, depth-first."""
    out = []
    for name, node in (states or {}).items():
        path = f"{prefix}{name}"
        out.append((path, name, node))
        if isinstance(node, dict) and node.get("states"):
            out += walk_states(node["states"], path + ".")
    return out


def action_names(v, problems=None, where=""):
    """Normalize an actions value (str | {type} | list of those) to [names]."""
    names = []
    for a in (v if isinstance(v, list) else [v] if v is not None else []):
        if isinstance(a, str):
            names.append(a)
        elif isinstance(a, dict) and isinstance(a.get("type"), str):
            names.append(a["type"])
        elif problems is not None:
            problems.append(f"unsupported action value {a!r}{' in ' + where if where else ''}"
                            f" (use a name string or {{\"type\": name}})")
    return names


def _norm(t, problems=None, where=""):
    """Normalize a transition value to dicts {target, guard, actions}."""
    for it in (t if isinstance(t, list) else [t]):
        if isinstance(it, str):
            yield {"target": it, "guard": None, "actions": []}
        elif isinstance(it, dict):
            tgt = it.get("target")
            if isinstance(tgt, list):
                if problems is not None:
                    problems.append(f"array transition target {tgt!r}{' in ' + where if where else ''}"
                                    f" (parallel targets are unsupported)")
                tgt = tgt[0] if tgt else None
            guard = it.get("guard")
            if guard is not None and not isinstance(guard, str) and problems is not None:
                problems.append(f"non-string guard {guard!r}{' in ' + where if where else ''}")
            yield {"target": tgt, "guard": guard if isinstance(guard, str) else None,
                   "actions": action_names(it.get("actions"), problems, where)}
        elif problems is not None:
            problems.append(f"unsupported transition value {it!r}{' in ' + where if where else ''}")


def transitions_of(node, problems=None, state=""):
    """All transitions defined on a state node, as flat dicts.

    kind is one of: on, after, always, onDone, onError, stateDone.
    `stateDone` is a compound state's own onDone (fires when its child final
    state is reached); it was previously invisible to every tool.
    """
    res = []
    for ev, t in (node.get("on") or {}).items():
        res += [{"kind": "on", "event": ev, **d}
                for d in _norm(t, problems, f"{state} on:{ev}")]
    for delay, t in (node.get("after") or {}).items():
        res += [{"kind": "after", "event": delay, **d}
                for d in _norm(t, problems, f"{state} after:{delay}")]
    if node.get("always"):
        res += [{"kind": "always", "event": "", **d}
                for d in _norm(node["always"], problems, f"{state} always")]
    if node.get("onDone"):
        res += [{"kind": "stateDone", "event": "", **d}
                for d in _norm(node["onDone"], problems, f"{state} onDone")]
    inv = node.get("invoke")
    if inv:
        for iv in (inv if isinstance(inv, list) else [inv]):
            for key in ("onDone", "onError"):
                if key in iv:
                    res += [{"kind": key, "event": iv.get("src", ""), **d}
                            for d in _norm(iv[key], problems, f"{state} invoke.{key}")]
    return res


def actions_of(node, problems=None, state=""):
    acc = set()
    for k in ("entry", "exit"):
        acc |= set(action_names(node.get(k), problems, f"{state} {k}"))
    for tr in transitions_of(node, None, state):
        acc |= set(tr["actions"])
    return acc


def invokes_of(node):
    inv = node.get("invoke")
    if not inv:
        return []
    return inv if isinstance(inv, list) else [inv]


def load_machine(path):
    """Load a machine JSON with a real error message instead of a traceback."""
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f), None
    except OSError as e:
        return None, f"cannot read {path}: {e}"
    except json.JSONDecodeError as e:
        return None, f"invalid JSON in {path}: line {e.lineno}: {e.msg}"


# --------------------------- structural lint ------------------------------

def lint_machine(m, base):
    """Structural lint of a parsed machine. Returns (errs, warns, notes, counts)."""
    errs, warns, notes = [], [], []
    counts = {"states": 0, "transitions": 0}

    for k in m:
        if k not in ROOT_KEYS:
            errs.append(f"{base}: unsupported root key {k!r} (supported: {', '.join(sorted(ROOT_KEYS))})")
    if not isinstance(m.get("states"), dict) or not m["states"]:
        errs.append(f"{base}: machine has no states")
        return errs, warns, notes, counts

    states = walk_states(m["states"])
    counts["states"] = len(states)
    paths = {p for p, _, _ in states}
    by_simple = {}
    for p, n, _ in states:
        by_simple.setdefault(n, []).append(p)

    def resolve(tgt, src_path):
        """Resolve a transition target to a state path, or (None, why)."""
        if tgt is None:
            return src_path, None  # internal (self) transition
        t = tgt.lstrip("#")
        if t in paths:
            return t, None
        cands = by_simple.get(t.split(".")[-1], [])
        if len(cands) == 1:
            return cands[0], None
        if len(cands) > 1:
            return None, f"ambiguous target {tgt!r} (candidates: {', '.join(sorted(cands))})"
        return None, f"dangling target {tgt!r}"

    # per-state structural checks
    problems = []
    for p, n, node in states:
        if not isinstance(node, dict):
            errs.append(f"{base}: state {p} is not an object")
            continue
        for k in node:
            if k not in STATE_KEYS:
                errs.append(f"{base}: unsupported key {k!r} in state {p}")
        stype = node.get("type")
        if stype not in STATE_TYPES:
            errs.append(f"{base}: unsupported state type {stype!r} in {p} "
                        f"(parallel/history are not in the supported subset)")
            continue
        trs = transitions_of(node, problems, p)
        counts["transitions"] += len(trs)
        is_final = stype == "final"
        if is_final and (trs or node.get("invoke")):
            errs.append(f"{base}: final state {p} declares transitions or an invoke; "
                        f"XState ignores them, so they are dead spec")
        if node.get("states") and not node.get("initial"):
            errs.append(f"{base}: compound state {p} has no initial")
        if node.get("initial") and not node.get("states"):
            errs.append(f"{base}: state {p} has initial but no child states")
        if not is_final and not node.get("states") and not trs:
            errs.append(f"{base}: dead-end non-final leaf state {p}")
        for iv in invokes_of(node):
            for k in iv:
                if k not in INVOKE_KEYS:
                    errs.append(f"{base}: unsupported invoke key {k!r} in state {p}")
            if "onError" not in iv:
                errs.append(f"{base}: invoke {iv.get('src')!r} in {p} has no onError")
        if node.get("invoke") and "after" not in node:
            errs.append(f"{base}: invoking state {p} has no after/timeout")
        action_names(node.get("entry"), problems, f"{p} entry")
        action_names(node.get("exit"), problems, f"{p} exit")

        # branch-list shape: nothing after an unguarded branch is reachable
        def check_shadow(label, t):
            branches = list(_norm(t))
            for i, b in enumerate(branches[:-1]):
                if b["guard"] is None:
                    errs.append(f"{base}: state {p} {label} branch {i + 1} is unguarded "
                                f"but not last; later branches are unreachable")

        for ev, t in (node.get("on") or {}).items():
            check_shadow(f"on:{ev}", t)
        for delay, t in (node.get("after") or {}).items():
            check_shadow(f"after:{delay}", t)
        if node.get("onDone"):
            check_shadow("onDone", node["onDone"])
        for iv in invokes_of(node):
            for key in ("onDone", "onError"):
                if key in iv:
                    check_shadow(f"invoke.{key}", iv[key])
        if node.get("always"):
            check_shadow("always", node["always"])
            branches = list(_norm(node["always"]))
            fully_guarded = all(b["guard"] for b in branches)
            has_escape = bool(node.get("after")) or bool(node.get("on")) or bool(node.get("invoke"))
            if fully_guarded and not has_escape:
                just = node.get("_exhaustive")
                if isinstance(just, str) and just.strip():
                    notes.append(f"{base}: state {p} always-list is fully guarded; liveness "
                                 f"rests on the declared exhaustiveness: {just.strip()}")
                else:
                    errs.append(f"{base}: state {p} has a fully guarded always-list and no "
                                f"unguarded escape (after/on/invoke); if no guard is true the "
                                f"machine is stuck. Add an unguarded fallback branch, or an "
                                f"_exhaustive note stating why the guards are total")

        for tr in trs:
            dest, why = resolve(tr["target"], p)
            if why:
                errs.append(f"{base}: {why} from {p} ({tr['kind']}:{tr['event']})")

    errs += [f"{base}: {pr}" for pr in problems]

    # event completeness: every RESTING state (top-level, non-final, no invoke,
    # no always: it sits waiting for external events) must handle or explicitly
    # ignore every event the machine reacts to. Transient states (invoke/always,
    # or lowerCamel overlay) resolve internally before an external event is
    # processed; final states reject structurally.
    all_events = set()
    for p, _, node in states:
        if isinstance(node, dict):
            all_events |= set((node.get("on") or {}).keys())
    for p, n, node in states:
        if "." in p or not isinstance(node, dict):
            continue
        if not n[:1].isupper() or node.get("type") == "final" or node.get("states"):
            continue
        if node.get("invoke") or node.get("always"):
            continue
        ignores = node.get("_ignores") or {}
        if not isinstance(ignores, dict) or not all(
                isinstance(v, str) and v.strip() for v in ignores.values()):
            errs.append(f"{base}: state {p} _ignores must map event names to reason strings")
            ignores = ignores if isinstance(ignores, dict) else {}
        for ev in sorted(ignores):
            if ev in (node.get("on") or {}):
                errs.append(f"{base}: state {p} both handles and ignores event {ev!r}")
        for ev in sorted(all_events):
            if ev not in (node.get("on") or {}) and ev not in ignores:
                errs.append(f"{base}: state {p} neither handles nor explicitly ignores "
                            f"event {ev!r} (add it to on: or to _ignores: with a reason)")

    # initial + reachability (hierarchical: entering a compound state enters its
    # initial child; a reached child implies its ancestors are active)
    init = m.get("initial")
    if init not in (m.get("states") or {}):
        errs.append(f"{base}: initial {init!r} is not a top-level state")
    else:
        node_of = {p: node for p, _, node in states}
        reached = set()

        def enter(p):
            if p in reached:
                return
            reached.add(p)
            node = node_of.get(p, {})
            if node.get("states") and node.get("initial"):
                child = f"{p}.{node['initial']}"
                if child in paths:
                    enter(child)

        enter(init)
        frontier = True
        while frontier:
            frontier = False
            active = set()
            for p in reached:
                active.add(p)
                # ancestors of a reached state are active too
                while "." in p:
                    p = p.rsplit(".", 1)[0]
                    active.add(p)
            for p in active:
                for tr in transitions_of(node_of.get(p, {})):
                    dest, why = resolve(tr["target"], p)
                    if dest and dest not in reached:
                        enter(dest)
                        frontier = True
        for p, n, _ in states:
            if p not in reached and not any(p.startswith(r + ".") or r.startswith(p + ".")
                                            for r in reached):
                errs.append(f"{base}: unreachable state {p}")

    return errs, warns, notes, counts


# ------------------------- matrix reconciliation --------------------------

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


def find_col(header, *names):
    for i, h in enumerate(header):
        hl = h.lower()
        if any(n in hl for n in names):
            return i
    return None


def _clean_cell(cell):
    """Strip backticks and parenthetical annotations from a matrix cell."""
    cell = cell.replace("`", "")
    cell = re.sub(r"\([^)]*\)", "", cell)
    return cell.strip()


def machine_transition_rows(m):
    """Canonical (source, trigger, guard, target, actions) rows from a machine.

    guard is the guard name, or None for an unguarded branch; `fallback` is True
    when the unguarded branch follows guarded ones (the matrix may write it as
    `!firstGuard`, `(else)`, or `-`). target is the resolved simple state name,
    or `(internal)` for a targetless transition.
    """
    states = walk_states(m.get("states"))
    rows = []
    for p, n, node in states:
        groups = {}
        for tr in transitions_of(node):
            trig = {"on": tr["event"],
                    "after": f"after {tr['event']}",
                    "always": "always",
                    "stateDone": "onDone",
                    "onDone": "invoke onDone",
                    "onError": "invoke onError"}[tr["kind"]]
            groups.setdefault(trig, []).append(tr)
        for trig, trs in groups.items():
            for i, tr in enumerate(trs):
                tgt = tr["target"].lstrip("#").split(".")[-1] if tr["target"] else "(internal)"
                rows.append({"source": n, "trigger": trig, "guard": tr["guard"],
                             "fallback": tr["guard"] is None and i > 0,
                             "first_guard": trs[0]["guard"],
                             "target": tgt, "actions": tuple(tr["actions"])})
    return rows


def matrix_transition_rows(text):
    """Parse the transition-matrix table of a matrix.md into canonical rows.

    Returns (rows, error). rows is None if the file has no transition table
    (legitimate when the generated oracle is the only transition artifact).
    """
    for header, rows in parse_md_tables(text):
        joined = " ".join(h.lower() for h in header)
        if "source" in joined and "target" in joined and "actions" in joined:
            si = find_col(header, "source")
            ei = find_col(header, "event", "trigger")
            gi = find_col(header, "guard")
            ti = find_col(header, "target")
            ai = find_col(header, "actions")
            if None in (si, ei, gi, ti, ai):
                return None, ("transition table found but a required column is missing "
                              "(need source, event/trigger, guard, target, actions)")
            out = []
            for r in rows:
                if len(r) <= max(si, ei, gi, ti, ai):
                    return None, f"transition table row has too few cells: {r!r}"
                if "(final)" in r[ei] or "(final)" in r[ti] or "(any event)" in r[ei]:
                    # documentation row for a final state (entry action or the
                    # structural any-event rejection), not a transition; the
                    # generated oracle's entry/exit table owns entry actions
                    continue
                src = _clean_cell(r[si])
                trig = _clean_cell(r[ei])
                guard = _clean_cell(r[gi])
                tgt_raw = r[ti]
                tgt = "(internal)" if "(internal)" in tgt_raw else _clean_cell(tgt_raw)
                acts = _clean_cell(r[ai])
                actions = tuple(a.strip() for a in acts.split(",") if a.strip()) \
                    if acts and acts != "-" else ()
                out.append({"source": src, "trigger": trig, "guard": guard,
                            "target": tgt, "actions": actions})
            return out, None
    return None, None


def _guard_matches(mrow, cell):
    """Does the matrix guard cell describe this machine branch?"""
    if mrow["guard"]:
        return cell == mrow["guard"]
    if mrow["fallback"]:
        accepted = {"-", "(else)", "else", ""}
        if mrow["first_guard"]:
            accepted.add(f"!{mrow['first_guard']}")
        return cell in accepted
    return cell in {"-", ""}


def reconcile_matrix(m, matrix_text, base):
    """Structural, bidirectional machine <-> matrix reconciliation."""
    drift = []
    mrows = machine_transition_rows(m)
    xrows, err = matrix_transition_rows(matrix_text)
    if err:
        return [f"{base}: {err}"], 0
    if xrows is None:
        return [], 0  # no transition table; the generated oracle covers transitions

    unmatched = list(range(len(xrows)))

    def take(pred):
        for k in list(unmatched):
            if pred(xrows[k]):
                unmatched.remove(k)
                return xrows[k]
        return None

    def trig_eq(machine_trig, cell):
        return cell == machine_trig or cell == f"on:{machine_trig}"

    for mr in mrows:
        hit = take(lambda x: x["source"] == mr["source"]
                   and trig_eq(mr["trigger"], x["trigger"])
                   and _guard_matches(mr, x["guard"])
                   and x["target"] == mr["target"]
                   and x["actions"] == mr["actions"])
        if hit is None:
            g = mr["guard"] or ("else" if mr["fallback"] else "-")
            drift.append(f"{base}: machine transition has no matrix row: "
                         f"{mr['source']} --{mr['trigger']} [{g}]--> {mr['target']} "
                         f"/ {', '.join(mr['actions']) or '-'}")
    for k in unmatched:
        x = xrows[k]
        drift.append(f"{base}: matrix row has no machine transition: "
                     f"{x['source']} --{x['trigger']} [{x['guard'] or '-'}]--> {x['target']} "
                     f"/ {', '.join(x['actions']) or '-'}")
    return drift, len(mrows)


def namedunit_names(matrix_text):
    """Names declared in the named-unit contract table (guards, actions, actors)."""
    names = set()
    for header, rows in parse_md_tables(matrix_text):
        hl = " ".join(h.lower() for h in header)
        if "signature" in hl or ("name" in hl and "kind" in hl):
            ni = find_col(header, "name") or 0
            for r in rows:
                cell = r[ni] if ni < len(r) else ""
                names |= set(re.findall(rf"`({IDENT})`", cell))
                if "`" not in cell:
                    names |= set(re.findall(rf"\b({IDENT})\b", cell))
    return names


def machine_unit_names(m):
    """Every guard, action, and actor name the machine references."""
    guards, actions, actors = set(), set(), set()
    for p, _, node in walk_states(m.get("states")):
        for tr in transitions_of(node):
            if tr["guard"]:
                guards.add(tr["guard"])
            actions |= set(tr["actions"])
        actions |= set(action_names(node.get("entry"))) | set(action_names(node.get("exit")))
        for iv in invokes_of(node):
            if isinstance(iv.get("src"), str):
                actors.add(iv["src"])
    return guards, actions, actors


# ------------------------------- CLI --------------------------------------

def lint(path):
    """Back-compatible entry: returns (n_states, errs, warns, drift)."""
    m, err = load_machine(path)
    base = os.path.basename(path)
    if err:
        return 0, [err], [], []
    errs, warns, notes, counts = lint_machine(m, base)
    drift = []
    mx = path.replace(".machine.json", ".matrix.md")
    if os.path.exists(mx):
        with open(mx, encoding="utf-8") as f:
            drift, _ = reconcile_matrix(m, f.read(), base)
    else:
        warns.append(f"{base}: no matrix file; named-unit contracts are unchecked "
                     f"(the generated oracle still covers transitions)")
    return counts["states"], errs, warns + notes, drift


def main():
    mdir = sys.argv[1] if len(sys.argv) > 1 else "."
    files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    if not files:
        print(f"ERROR  no *.machine.json under {mdir}: nothing to lint is a failure, not a pass")
        sys.exit(1)
    total = 0
    for f in files:
        n, errs, warns, drift = lint(f)
        print(f"== {os.path.basename(f)}: {n} states ==")
        for e in errs:
            print(f"  ERROR  {e}")
        for d in drift:
            print(f"  DRIFT  {d}")
        for w in warns:
            print(f"  warn   {w}")
        if not (errs or drift or warns):
            print("  ok")
        total += len(errs) + len(drift)
    print(f"\n{total} error/drift finding(s) across {len(files)} machine(s)")
    sys.exit(1 if total else 0)


if __name__ == "__main__":
    main()
