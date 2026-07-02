#!/usr/bin/env python3
"""machine_lint: deterministic checks for machinery Phase-3 XState machines.

No LLM, no dependencies. Two layers:
  1. Structural lint (symbolic, over the finite state graph): valid JSON, every
     transition target resolves, no dead-end leaf states, and every invoke has
     both an onError branch and an after/timeout. Reachability is advisory.
  2. JSON <-> matrix reconciliation: every action the machine actually fires
     (including entry/exit actions) must appear in a row of the matrix oracle.
     This is the class of drift the LLM review missed in the go-crm dry run.

Exit non-zero on any ERROR or DRIFT finding, so it can gate Phase 3.

Usage: machine_lint.py <machines-dir>
"""
import json, sys, os, re, glob


def walk_states(states, prefix=""):
    out = []
    for name, node in (states or {}).items():
        path = f"{prefix}{name}"
        out.append((path, name, node))
        if node.get("states"):
            out += walk_states(node["states"], path + ".")
    return out


def _norm(t):
    for it in (t if isinstance(t, list) else [t]):
        if isinstance(it, str):
            yield {"target": it, "guard": None, "actions": []}
        elif isinstance(it, dict):
            acts = it.get("actions", [])
            acts = acts if isinstance(acts, list) else [acts]
            yield {"target": it.get("target"), "guard": it.get("guard"),
                   "actions": [a for a in acts if isinstance(a, str)]}


def transitions_of(node):
    res = []
    for ev, t in (node.get("on") or {}).items():
        res += [{"kind": "on", "event": ev, **d} for d in _norm(t)]
    for delay, t in (node.get("after") or {}).items():
        res += [{"kind": "after", "event": delay, **d} for d in _norm(t)]
    if node.get("always"):
        res += [{"kind": "always", "event": "", **d} for d in _norm(node["always"])]
    inv = node.get("invoke")
    if inv:
        for iv in (inv if isinstance(inv, list) else [inv]):
            for key in ("onDone", "onError"):
                if key in iv:
                    res += [{"kind": key, "event": iv.get("src", ""), **d} for d in _norm(iv[key])]
    return res


def actions_of(node):
    acc = set()
    for k in ("entry", "exit"):
        v = node.get(k)
        if v:
            acc |= set(v if isinstance(v, list) else [v])
    for tr in transitions_of(node):
        acc |= set(tr["actions"])
    return {a for a in acc if isinstance(a, str)}


def lint(path):
    errs, warns, drift = [], [], []
    m = json.load(open(path))
    states = walk_states(m.get("states"))
    paths = {p for p, _, _ in states}
    simple = {n for _, n, _ in states}

    def resolves(tgt):
        if tgt is None:
            return True
        t = tgt.lstrip("#")
        return t in paths or t in simple or t.split(".")[-1] in simple

    edges = {p: set() for p, _, _ in states}
    for p, _, node in states:
        for tr in transitions_of(node):
            if not resolves(tr["target"]):
                errs.append(f"dangling target {tr['target']!r} from {p} ({tr['kind']}:{tr['event']})")
            elif tr["target"]:
                t = tr["target"].lstrip("#")
                dest = t if t in paths else next((q for q, n, _ in states if n == t.split(".")[-1]), None)
                if dest:
                    edges[p].add(dest)

    init = m.get("initial")
    roots = [p for p, n, _ in states if n == init]
    seen, stack = set(), list(roots)
    while stack:
        x = stack.pop()
        if x not in seen:
            seen.add(x)
            stack += list(edges.get(x, ()))
    for p, n, _ in states:
        if "." not in p and p not in seen and p not in roots:
            warns.append(f"unreachable top-level state {p}")

    for p, _, node in states:
        is_final = node.get("type") == "final"
        if not is_final and "states" not in node and not transitions_of(node):
            errs.append(f"dead-end non-final leaf state {p}")
        if "invoke" in node:
            invs = node["invoke"]
            for iv in (invs if isinstance(invs, list) else [invs]):
                if "onError" not in iv:
                    errs.append(f"invoke {iv.get('src')!r} in {p} has no onError")
            if "after" not in node:
                errs.append(f"invoking state {p} has no after/timeout")

    # matrix reconciliation: actions fired by the machine must appear in a table row
    mx = path.replace(".machine.json", ".matrix.md")
    if os.path.exists(mx):
        rows = "\n".join(l for l in open(mx).read().splitlines() if l.lstrip().startswith("|"))
        row_tokens = set(re.findall(r"[a-z][a-zA-Z0-9]+", rows))
        fired = set()
        for _, _, node in states:
            fired |= actions_of(node)
        for a in sorted(fired):
            if a not in row_tokens:
                drift.append(f"action {a!r} is fired by the machine but appears in no matrix row")
    else:
        errs.append(f"no matrix file beside {os.path.basename(path)}")

    return len(states), errs, warns, drift


def main():
    mdir = sys.argv[1] if len(sys.argv) > 1 else "."
    total = 0
    for f in sorted(glob.glob(os.path.join(mdir, "*.machine.json"))):
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
    print(f"\n{total} error/drift finding(s)")
    sys.exit(1 if total else 0)


if __name__ == "__main__":
    main()
