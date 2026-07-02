#!/usr/bin/env python3
"""oracle_gen: generate the canonical transition oracle from a machine JSON.

Rung 1 of the correctness ladder. The machine JSON is the single source; the
transition matrix and the hard-TDD test oracle are GENERATED from it, not
co-authored beside it. That makes machine-vs-oracle drift impossible by
construction rather than something a checker has to detect after the fact.

Emits <M>.oracle.md next to each machine: a per-state entry/exit action table and
a numbered transition table where every row is exactly one test case (given
source + trigger + guard, expect target + actions). Deterministic; no LLM.

Usage: oracle_gen.py <machines-dir>
"""
import sys, os, json, glob

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of  # noqa: E402


def _fmt(v):
    if not v:
        return "-"
    return ", ".join(v if isinstance(v, list) else [v])


def generate(path):
    m = json.load(open(path))
    mid = m.get("id", os.path.splitext(os.path.basename(path))[0])
    states = walk_states(m.get("states"))
    tag = mid.upper()[:4]
    L = []
    L.append(f"# Generated transition oracle: `{mid}`")
    L.append("")
    L.append(f"Generated from `{os.path.basename(path)}` by tools/oracle_gen.py. DO NOT EDIT BY HAND.")
    L.append("Single source of truth for the hard-TDD transition tests: one transition row is one test case.")
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
    L.append("| test id | source | trigger | guard | target | actions |")
    L.append("|---|---|---|---|---|---|")
    i = 0
    for p, n, node in states:
        for tr in transitions_of(node):
            i += 1
            trig = f"{tr['kind']}:{tr['event']}" if tr.get("event") else tr["kind"]
            guard = tr.get("guard") or "-"
            target = tr["target"] or "(internal)"
            L.append(f"| T-{tag}-{i:02d} | {p} | {trig} | {guard} | {target} | {_fmt(tr['actions'])} |")
    L.append("")
    L.append(f"Total transitions (test cases): {i}")
    L.append("")
    return "\n".join(L)


def main():
    mdir = sys.argv[1] if len(sys.argv) > 1 else "."
    files = sorted(glob.glob(os.path.join(mdir, "*.machine.json")))
    if not files:
        print(f"no *.machine.json under {mdir}")
        sys.exit(1)
    for f in files:
        out = f.replace(".machine.json", ".oracle.md")
        open(out, "w").write(generate(f))
        n = generate(f).count("| T-")
        print(f"generated {os.path.basename(out)}  ({n} transition rows)")


if __name__ == "__main__":
    main()
