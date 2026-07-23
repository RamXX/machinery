#!/usr/bin/env python3
"""Adapter for the pii-flow external checker.

Bridges the machinery projection contract (schemas/projection.schema.json) and
evidence contract (schemas/evidence.schema.json) to a Soufflé Datalog program
(rules.dl) that decides whether a sensitive attribute can reach an export sink
without passing through a redactor.

Invocation (see examples/pii-flow/checkers.local.example.yaml):

    adapter.py <projection.json> <config.json> <evidence.json>

<config.json> is the manifest's `config` block, rendered to a standalone JSON
file by the caller. Standard library only: no third-party dependencies.
"""

import csv
import json
import subprocess
import sys
import tempfile
from pathlib import Path


def write_facts(facts_dir: Path, name: str, rows) -> None:
    """Write one Soufflé .facts file: tab-separated, one row per line.

    A relation with zero rows still needs its file present (Soufflé's .input
    directive fails on a missing file, not on an empty one), so this always
    creates the file even when `rows` is empty.
    """
    path = facts_dir / f"{name}.facts"
    with path.open("w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f, delimiter="\t", lineterminator="\n")
        for row in rows:
            writer.writerow(row)


def main() -> int:
    if len(sys.argv) != 4:
        print("usage: adapter.py <projection.json> <config.json> <evidence.json>", file=sys.stderr)
        return 2

    projection_path, config_path, out_path = (Path(a) for a in sys.argv[1:4])

    projection = json.loads(projection_path.read_text(encoding="utf-8"))
    config = json.loads(config_path.read_text(encoding="utf-8"))

    model = projection.get("model", {})
    entities = model.get("entities", [])
    relationships = model.get("relationships", [])

    attr_of_rows = [
        (entity["stable_id"], attr["stable_id"])
        for entity in entities
        for attr in entity.get("attributes", [])
    ]
    flows_rows = [(rel["from"], rel["to"]) for rel in relationships]
    sensitive_rows = [(s,) for s in config.get("sensitive", [])]
    sink_rows = [(s,) for s in config.get("sinks", [])]
    redacted_rows = [(r,) for r in config.get("redacted", [])]

    rules_dl = Path(__file__).resolve().parent / "rules.dl"

    with tempfile.TemporaryDirectory(prefix="pii-flow-facts-") as facts_dir_s, \
         tempfile.TemporaryDirectory(prefix="pii-flow-out-") as out_dir_s:
        facts_dir = Path(facts_dir_s)
        out_dir = Path(out_dir_s)

        write_facts(facts_dir, "attr_of", attr_of_rows)
        write_facts(facts_dir, "flows", flows_rows)
        write_facts(facts_dir, "sensitive", sensitive_rows)
        write_facts(facts_dir, "sink", sink_rows)
        write_facts(facts_dir, "redacted", redacted_rows)

        result = subprocess.run(
            ["souffle", str(rules_dl), "-F", str(facts_dir), "-D", str(out_dir)],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            sys.stderr.write(result.stdout)
            sys.stderr.write(result.stderr)
            return result.returncode

        leak_csv = out_dir / "leak.csv"
        leaks = []
        if leak_csv.exists():
            with leak_csv.open(encoding="utf-8", newline="") as f:
                for row in csv.reader(f, delimiter="\t"):
                    if row:
                        leaks.append(row[0])

    input_hash = projection["generated"]["input_hash"]
    verdict = "pass" if not leaks else "fail"
    enforced = config["enforces_invariant"]

    evidence = {
        "evidence_schema": "1.0",
        "checker": {"id": "pii-flow", "version": "souffle"},
        "input_hash": input_hash,
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
