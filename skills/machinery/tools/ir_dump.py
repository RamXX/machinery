#!/usr/bin/env python3
"""ir_dump: canonical IR serialization for differential parity testing.

Phase 2 of the migration proves the parser (ordering + traversal) matches before
anything is generated from it. This emits the parsed machine as a stable JSON:
  { "machine": <id>, "initial": <initial>,
    "states": [{"path","name","type","entry","exit"}, ...],
    "transitions": [{"state","kind","event","target","guard","actions"}, ...] }
traversed in source order (states depth-first, transitions in on/after/always/
stateDone/invoke.onDone/onError order). machinery ir-dump produces the same.
"""
import json, sys, os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of, load_machine, action_names  # noqa: E402


def dump(path):
    m, err = load_machine(path)
    if err:
        sys.exit(f"ir_dump: {err}")
    states = walk_states(m.get("states"))
    out = {
        "machine": m.get("id"),
        "initial": m.get("initial"),
        "states": [
            {
                "path": p, "name": n,
                "type": node.get("type") if isinstance(node, dict) else None,
                "entry": sorted(set(action_names(node.get("entry")))),
                "exit": sorted(set(action_names(node.get("exit")))),
            }
            for p, n, node in states if isinstance(node, dict)
        ],
        "transitions": [
            {
                "state": p,
                "kind": tr["kind"],
                "event": tr["event"],
                "target": tr["target"],
                "guard": tr["guard"],
                "actions": list(tr["actions"]),
            }
            for p, n, node in states
            for tr in transitions_of(node)
        ],
    }
    return out


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit("usage: ir_dump.py <machine.json>")
    print(json.dumps(dump(sys.argv[1]), indent=2, ensure_ascii=False))
