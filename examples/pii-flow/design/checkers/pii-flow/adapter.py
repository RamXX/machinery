#!/usr/bin/env python3
"""Hermetic adapter for the pii-flow external checker.

Bridges the machinery projection contract (schemas/projection.schema.json) and
evidence contract (schemas/evidence.schema.json) to the same fixed-point
semantics documented by rules.dl: whether a sensitive attribute can reach an
export sink without passing through a redactor. It runs from a digest-pinned
OCI Python userspace; there is no host interpreter/module/native-library input.

Invocation (see examples/pii-flow/checkers.local.example.yaml):

    adapter.py <projection.json> <config.json> <evidence.json> [rules.dl]

<config.json> is the manifest's `config` block, rendered to a standalone JSON
file by the caller. Standard library only: no third-party dependencies.
"""

import json
import os
import sys
from pathlib import Path


def main() -> int:
    if len(sys.argv) not in (4, 5):
        print("usage: adapter.py <projection.json> <config.json> <evidence.json> [rules.dl]", file=sys.stderr)
        return 2

    projection_path, config_path, out_path = (Path(a) for a in sys.argv[1:4])

    projection = json.loads(projection_path.read_text(encoding="utf-8"))
    config = json.loads(config_path.read_text(encoding="utf-8"))

    model = projection.get("model", {})
    entities = model.get("entities", [])
    relationships = model.get("relationships", [])

    attr_of = [
        (entity["stable_id"], attr["stable_id"])
        for entity in entities
        for attr in entity.get("attributes", [])
    ]
    flows = [(rel["from"], rel["to"]) for rel in relationships]
    sensitive = set(config.get("sensitive", []))
    sinks = set(config.get("sinks", []))
    redacted = set(config.get("redacted", []))

    rules_dl = Path(sys.argv[4]) if len(sys.argv) == 5 else Path(__file__).resolve().parent / "rules.dl"
    if "leak(E) :- tainted(E), sink(E)." not in rules_dl.read_text(encoding="utf-8"):
        print("rules.dl does not carry the pinned pii-flow leak contract", file=sys.stderr)
        return 2

    tainted = {entity for entity, attr in attr_of if attr in sensitive}
    changed = True
    while changed:
        changed = False
        for source, target in flows:
            if source in tainted and target not in redacted and target not in tainted:
                tainted.add(target)
                changed = True
    leaks = sorted(tainted & sinks)

    runtime_closure = os.environ.get("MACHINERY_CHECKER_RUNTIME_CLOSURE", "")
    if not runtime_closure.startswith("sha256:") or len(runtime_closure) != 71:
        print("MACHINERY_CHECKER_RUNTIME_CLOSURE is missing or malformed", file=sys.stderr)
        return 2

    input_hash = projection["generated"]["input_hash"]
    verdict = "pass" if not leaks else "fail"
    enforced = config["enforces_invariant"]

    evidence = {
        "evidence_schema": "1.0",
        "checker": {"id": "pii-flow", "version": "pii-flow-fixed-point-1"},
        "input_hash": input_hash,
        "runtime_closure": runtime_closure,
        "verdict": verdict,
        "coverage": [
            {"element": "inv:" + enforced, "verdict": verdict},
        ],
        "findings": [
            {
                "severity": "blocking",
                "code": "leak",
                "element": sink_id,
                "message": "sensitive data reaches export sink without redaction",
            }
            for sink_id in leaks
        ],
    }

    out_path.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    sys.exit(main())
