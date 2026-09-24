#!/usr/bin/env python3
"""Translation-only adapter for the pii-flow external checker.

The verdict is computed by Souffle, never here. This file only translates
between machinery's contracts and Souffle's native formats:

  1. projection.json (schemas/projection.schema.json, 1.0) plus the manifest's
     opaque config block  ->  one tab-separated .facts file per input relation;
  2. souffle --no-preprocessor -F <facts> -D <out> rules.dl;
  3. Souffle's output relations (leak.csv, tainted.csv)  ->  evidence.json
     (schemas/evidence.schema.json) plus a byte-stable trace below generated/.

Invocation (see examples/pii-flow/checkers.local.example.yaml):

    adapter.py run    <projection.json> <config.json> <evidence.json> <rules.dl>
    adapter.py verify <projection.json> <config.json> <evidence.json> <rules.dl>

`run` writes <evidence.json> and generated/<trace> beside it. `verify` re-runs
Souffle and requires the committed evidence and trace at <evidence.json> to be
byte-identical to what the engine derives now; it writes nothing.

Fail closed: any inconsistency (an unexpected projection shape or layer, a
fact value holding a tab/CR/LF, souffle missing, souffle exiting non-zero or
printing anything, a declared output relation Souffle did not write, a
malformed output row) exits non-zero with one diagnostic on stderr and writes
no evidence. Standard library only.
"""

import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

CHECKER_ID = "pii-flow"
CHECKER_VERSION = "pii-flow-souffle-1"
TRACE_REF = "generated/souffle-outputs.json"

# The only projection this adapter understands: the 1.0 contract carrying
# exactly the three layers the manifest includes. A 2.0 projection, a `layers`
# object, or any other top-level key means the manifest and the adapter
# disagree about the input, and silently ignoring the extra layer would decide
# the invariant over a design the checker never read. That is rejected.
PROJECTION_SCHEMA = "1.0"
PROJECTION_LAYERS = {"model", "invariants", "relationships"}
PROJECTION_KEYS = {
    "projection_schema", "machinery_version", "design_id", "checker_id",
    "manifest_hash", "include", "model", "generated",
}
MODEL_KEYS = {"entities", "invariants", "relationships"}

INPUT_RELATIONS = ("attr_of", "flows", "sensitive", "sink", "redacted")
# Every relation rules.dl declares with .output, and its arity. Souffle writes
# <name>.csv for each, empty when nothing is derived; a missing or extra file
# means the rules and this adapter disagree.
OUTPUT_RELATIONS = {"leak": 1, "tainted": 1}

SOUFFLE_TIMEOUT_SECONDS = 45


class AdapterError(Exception):
    """A fail-closed condition: reported on stderr, no evidence written."""


def fact_value(value, where):
    if not isinstance(value, str) or value == "":
        raise AdapterError(f"{where}: fact value must be a non-empty string, got {value!r}")
    if any(c in value for c in "\t\r\n"):
        raise AdapterError(f"{where}: fact value {value!r} holds a tab, carriage return, or newline; the facts format has no escape")
    return value


def load_projection(path):
    projection = json.loads(Path(path).read_text(encoding="utf-8"))
    if not isinstance(projection, dict):
        raise AdapterError("projection is not a JSON object")
    if projection.get("projection_schema") != PROJECTION_SCHEMA:
        raise AdapterError(f"projection_schema {projection.get('projection_schema')!r} is not {PROJECTION_SCHEMA!r}; this adapter reads only the 1.0 contract")
    extra = sorted(set(projection) - PROJECTION_KEYS)
    if extra:
        raise AdapterError(f"projection carries layers or keys this adapter does not read: {', '.join(extra)}")
    include = projection.get("include")
    if not isinstance(include, list) or set(include) != PROJECTION_LAYERS or len(include) != len(PROJECTION_LAYERS):
        raise AdapterError(f"projection include {include!r} is not exactly {sorted(PROJECTION_LAYERS)}")
    if projection.get("checker_id") != CHECKER_ID:
        raise AdapterError(f"projection checker_id {projection.get('checker_id')!r} is not {CHECKER_ID!r}")
    model = projection.get("model")
    if not isinstance(model, dict) or set(model) - MODEL_KEYS:
        raise AdapterError("projection model is missing or carries unexpected keys")
    input_hash = (projection.get("generated") or {}).get("input_hash", "")
    if not (isinstance(input_hash, str) and input_hash.startswith("sha256:") and len(input_hash) == 71):
        raise AdapterError("projection generated.input_hash is missing or malformed; run `machinery project`")
    return projection


def build_facts(projection, config):
    model = projection["model"]
    attr_of = set()
    for entity in model.get("entities", []):
        eid = fact_value(entity.get("stable_id"), "model.entities[].stable_id")
        for attr in entity.get("attributes", []):
            attr_of.add((eid, fact_value(attr.get("stable_id"), f"{eid} attributes[].stable_id")))
    flows = {
        (fact_value(rel.get("from"), "model.relationships[].from"), fact_value(rel.get("to"), "model.relationships[].to"))
        for rel in model.get("relationships", [])
    }

    def config_list(key):
        values = config.get(key)
        if not isinstance(values, list) or not values:
            raise AdapterError(f"config.{key} must be a non-empty list of stable ids")
        return {(fact_value(v, f"config.{key}"),) for v in values}

    return {
        "attr_of": attr_of,
        "flows": flows,
        "sensitive": config_list("sensitive"),
        "sink": config_list("sinks"),
        "redacted": config_list("redacted"),
    }


def write_facts(facts, directory):
    for name in INPUT_RELATIONS:
        lines = sorted("\t".join(row) for row in facts[name])
        (directory / f"{name}.facts").write_text("".join(line + "\n" for line in lines), encoding="utf-8")


def run_souffle(rules, facts_dir, out_dir):
    souffle = shutil.which("souffle")
    if souffle is None:
        raise AdapterError("souffle executable not found on PATH; the checker image must provide Souffle")
    try:
        result = subprocess.run(
            [souffle, "--no-preprocessor", "-F", str(facts_dir), "-D", str(out_dir), str(rules)],
            capture_output=True, timeout=SOUFFLE_TIMEOUT_SECONDS, check=False,
        )
    except subprocess.TimeoutExpired as exc:
        raise AdapterError(f"souffle did not finish within {SOUFFLE_TIMEOUT_SECONDS}s") from exc
    diagnostics = (result.stdout + result.stderr).decode("utf-8", "replace").strip()
    if result.returncode != 0:
        raise AdapterError(f"souffle exited {result.returncode}: {diagnostics}")
    if diagnostics:
        raise AdapterError(f"souffle printed diagnostics on an exit-zero run; warnings are not dropped: {diagnostics}")
    version = subprocess.run([souffle, "--version"], capture_output=True, timeout=SOUFFLE_TIMEOUT_SECONDS, check=False)
    for line in version.stdout.decode("utf-8", "replace").splitlines():
        if line.startswith("Version: "):
            return line[len("Version: "):].split()[0]
    raise AdapterError("souffle --version reported no version")


def read_outputs(out_dir):
    present = sorted(p.name for p in out_dir.iterdir())
    expected = sorted(f"{name}.csv" for name in OUTPUT_RELATIONS)
    missing = sorted(set(expected) - set(present))
    if missing:
        raise AdapterError(f"souffle wrote no output for declared relation(s): {', '.join(missing)}")
    extra = sorted(set(present) - set(expected))
    if extra:
        raise AdapterError(f"souffle wrote output this adapter does not map: {', '.join(extra)}")
    outputs = {}
    for name, arity in sorted(OUTPUT_RELATIONS.items()):
        rows = set()
        for line in (out_dir / f"{name}.csv").read_text(encoding="utf-8").splitlines():
            row = tuple(line.split("\t"))
            if len(row) != arity or any(v == "" for v in row):
                raise AdapterError(f"{name}.csv row {line!r} does not have arity {arity}")
            rows.add(row)
        outputs[name] = sorted(rows)
    return outputs


def derive(projection_path, config_path, rules_path):
    projection = load_projection(projection_path)
    config = json.loads(Path(config_path).read_text(encoding="utf-8"))
    if not isinstance(config, dict):
        raise AdapterError("config is not a JSON object")
    enforced = config.get("enforces_invariant")
    invariant_ids = {inv.get("stable_id") for inv in projection["model"].get("invariants", [])}
    if not isinstance(enforced, str) or "inv:" + enforced not in invariant_ids:
        raise AdapterError(f"config.enforces_invariant {enforced!r} is not an invariant in the projection")
    runtime_closure = os.environ.get("MACHINERY_CHECKER_RUNTIME_CLOSURE", "")
    if not runtime_closure.startswith("sha256:") or len(runtime_closure) != 71:
        raise AdapterError("MACHINERY_CHECKER_RUNTIME_CLOSURE is missing or malformed")
    rules = Path(rules_path)
    rules_bytes = rules.read_bytes()

    facts = build_facts(projection, config)
    with tempfile.TemporaryDirectory(prefix="pii-flow-") as scratch:
        facts_dir = Path(scratch, "facts")
        out_dir = Path(scratch, "out")
        facts_dir.mkdir()
        out_dir.mkdir()
        write_facts(facts, facts_dir)
        engine_version = run_souffle(rules, facts_dir, out_dir)
        outputs = read_outputs(out_dir)

    leaks = [row[0] for row in outputs["leak"]]
    verdict = "fail" if leaks else "pass"
    detail = (
        f"souffle derived {len(leaks)} leak tuple(s)" if leaks
        else "souffle derived no leak tuple: no sensitive attribute reaches a sink unredacted"
    )
    evidence = {
        "evidence_schema": "1.0",
        "checker": {"id": CHECKER_ID, "version": CHECKER_VERSION},
        "input_hash": projection["generated"]["input_hash"],
        "runtime_closure": runtime_closure,
        "verdict": verdict,
        "coverage": [{"element": "inv:" + enforced, "verdict": verdict, "detail": detail}],
        "findings": [
            {
                "severity": "blocking",
                "code": "leak",
                "element": sink,
                "message": "sensitive data reaches export sink without redaction",
                "locator": "rules.dl:leak",
            }
            for sink in leaks
        ],
        "attestation": {
            "engine": "souffle",
            "engine_version": engine_version,
            "rules_sha256": "sha256:" + hashlib.sha256(rules_bytes).hexdigest(),
        },
        "trace_ref": TRACE_REF,
    }
    trace = {
        "engine": "souffle",
        "engine_version": engine_version,
        "outputs": {name: [list(row) for row in rows] for name, rows in outputs.items()},
    }
    return (
        (json.dumps(evidence, indent=2) + "\n").encode("utf-8"),
        (json.dumps(trace, indent=2, sort_keys=True) + "\n").encode("utf-8"),
    )


def main(argv):
    if len(argv) != 6 or argv[1] not in ("run", "verify"):
        print("usage: adapter.py run|verify <projection.json> <config.json> <evidence.json> <rules.dl>", file=sys.stderr)
        return 2
    mode, projection_path, config_path, evidence_path, rules_path = argv[1:]
    evidence_path = Path(evidence_path)
    trace_path = evidence_path.parent / TRACE_REF
    try:
        evidence, trace = derive(projection_path, config_path, rules_path)
        if mode == "run":
            trace_path.parent.mkdir(parents=True, exist_ok=True)
            trace_path.write_bytes(trace)
            evidence_path.write_bytes(evidence)
            return 0
        if evidence_path.read_bytes() != evidence:
            raise AdapterError("committed evidence differs from what souffle derives now")
        if trace_path.read_bytes() != trace:
            raise AdapterError(f"committed trace {TRACE_REF} differs from what souffle derives now")
        return 0
    except (AdapterError, OSError, ValueError) as exc:
        print(f"pii-flow adapter: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
