#!/usr/bin/env python3
"""oracle_gen: generate the canonical transition oracle from a machine JSON.

Rung 1 of the correctness ladder. The machine JSON is the single source; the
transition oracle is GENERATED from it, not co-authored beside it. machinery_check
G3 regenerates this file in memory and diffs it against the committed copy, so a
stale oracle is caught as DRIFT rather than assumed fresh.

Every row is exactly one test case (given source + trigger + guard, expect
target + actions) and carries two identifiers:

  test id    T-<TAG>-<nn>: sequential, for humans reading the table.
  stable id  <TAG>-<hex6>: content-derived from (machine, source, trigger, guard),
             so it survives unrelated row insertions and deletions. Tests MUST key
             on the stable id; row numbers renumber on the first design change.
             A stable id disappears only when its transition's stimulus changes,
             which correctly reads as "that test case is gone".

Deterministic; no LLM.

Usage: oracle_gen.py <machines-dir>
"""
import hashlib
import sys, os, json, glob

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of, load_machine, action_names  # noqa: E402


def _fmt(v):
    names = action_names(v)
    return ", ".join(names) if names else "-"


def stable_id(tag, source, trig, guard):
    h = hashlib.sha256(f"{tag}|{source}|{trig}|{guard or ''}".encode()).hexdigest()[:6]
    return f"{tag}-{h}"


def render(m, source_name):
    """Render the oracle markdown for a parsed machine. Pure."""
    mid = m.get("id", os.path.splitext(os.path.basename(source_name))[0])
    states = walk_states(m.get("states"))
    tag = mid.upper()[:4]
    L = []
    L.append(f"# Generated transition oracle: `{mid}`")
    L.append("")
    L.append(f"Generated from `{os.path.basename(source_name)}` by tools/oracle_gen.py. DO NOT EDIT BY HAND.")
    L.append("Single source of truth for the hard-TDD transition tests: one transition row is one")
    L.append("test case. Key tests on the STABLE id, not the row number; row numbers renumber when")
    L.append("the design changes, stable ids do not.")
    L.append("")
    L.append("## State entry / exit actions")
    L.append("")
    L.append("| state | kind | entry | exit |")
    L.append("|---|---|---|---|")
    for p, n, node in states:
        if "." in p:
            continue
        L.append(f"| {p} | {node.get('type', 'atomic')} | {_fmt(node.get('entry'))} | {_fmt(node.get('exit'))} |")
    L.append("")
    L.append("## Transitions")
    L.append("")
    L.append("| test id | stable id | source | trigger | guard | target | actions |")
    L.append("|---|---|---|---|---|---|---|")
    i = 0
    seen = {}
    for p, n, node in states:
        for tr in transitions_of(node):
            i += 1
            trig = f"{tr['kind']}:{tr['event']}" if tr.get("event") else tr["kind"]
            guard = tr.get("guard") or "-"
            target = tr["target"] or "(internal)"
            sid = stable_id(tag, p, trig, tr.get("guard"))
            if sid in seen:
                # two branches with the same stimulus: lint flags the shadowing;
                # disambiguate here so the oracle never emits a duplicate key
                seen[sid] += 1
                sid = f"{sid}.{seen[sid]}"
            else:
                seen[sid] = 1
            L.append(f"| T-{tag}-{i:02d} | {sid} | {p} | {trig} | {guard} | {target} | {_fmt(tr['actions'])} |")
    L.append("")
    L.append(f"Total transitions (test cases): {i}")
    L.append("")
    return "\n".join(L)


def generate(path):
    """Render the oracle for a machine file. Raises SystemExit on a bad machine."""
    m, err = load_machine(path)
    if err:
        sys.exit(f"oracle_gen: {err}")
    return render(m, path)


def main():
    mdir = sys.argv[1] if len(sys.argv) > 1 else "."
    files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    if not files:
        print(f"no *.machine.json under {mdir}")
        sys.exit(1)
    for f in files:
        out = f.replace(".machine.json", ".oracle.md")
        body = generate(f)
        with open(out, "w", encoding="utf-8") as fh:
            fh.write(body)
        print(f"generated {os.path.basename(out)}  ({body.count('| T-')} transition rows)")


if __name__ == "__main__":
    main()
