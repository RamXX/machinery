#!/usr/bin/env python3
"""tla_gen: translate a machinery machine (the IR) into a TLA+ control-flow model.

Rung 3 of the correctness ladder, first cut. Emits a TLA+ module plus a TLC config
from a machine JSON so the machine can be model-checked exhaustively (TLC) rather
than reviewed by eye. The model is the control graph plus a bounded retry counter
so liveness is decidable:

  - `st` ranges over the machine states; `rc` is a bounded retry counter.
  - Guards are over-approximated (both branches of a guarded event are possible),
    which is SOUND for the safety and liveness properties below: if they hold over
    the larger behavior set, they hold for the real guarded subset.
  - The standard persist-retry overlay (a state with a guarded `always` and an
    `after`) is modeled faithfully with the counter, which is what makes the
    liveness property provable.

Generated properties:
  TypeOK               (safety) states are valid; the retry counter never exceeds its bound.
  Live_OverlayResolves (liveness) from any transient overlay state the machine
                       always eventually returns to a resting domain state; it can
                       never get stuck retrying or half-persisted.
TLC also checks deadlock-freedom by default.

The data-refined invariants (stage-forward, atomicity of the persisted value) need
the action semantics, which live in the code, not the machine config; those are a
hand-annotated refinement on top of this generated skeleton, and the next step.

Usage: tla_gen.py <machine.json> [out-dir]
"""
import sys, os, json

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of  # noqa: E402


def classify(states):
    domain, overlay = set(), set()
    for _, n, node in states:
        if node.get("on") or node.get("type") == "final":
            domain.add(n)
        else:
            overlay.add(n)
    return domain, overlay


def retry_state(states):
    for _, n, node in states:
        if node.get("always") and node.get("after"):
            return n, node
    return None, None


def _first_target(x):
    for it in (x if isinstance(x, list) else [x]):
        if isinstance(it, dict) and it.get("target"):
            return it["target"]
        if isinstance(it, str):
            return it
    return None


def _simple(t):
    return t.lstrip("#").split(".")[-1] if t else None


def generate(path):
    m = json.load(open(path))
    mid = m.get("id", "machine").capitalize()
    states = [(p, n, node) for p, n, node in walk_states(m.get("states")) if "." not in p]
    names = [n for _, n, _ in states]
    domain, overlay = classify(states)
    rname, rnode = retry_state(states)
    initial = m.get("initial")

    dom_actions, ovl_actions, defs, comments = [], [], [], []
    idx = 0
    for _, n, node in states:
        if n == rname:
            continue
        for tr in transitions_of(node):
            idx += 1
            tgt = _simple(tr["target"]) or n
            if tgt in domain:
                rc = "0"
            elif n in domain and tgt in overlay:
                rc = "0"
            else:
                rc = "rc"
            name = f"T{idx}"
            defs.append(f'{name} == st = "{n}" /\\ st\' = "{tgt}" /\\ rc\' = {rc}')
            trig = f'{tr["kind"]}:{tr["event"]}' if tr.get("event") else tr["kind"]
            comments.append(f"  \\* {name}: {n} -{trig}-> {tgt}")
            (dom_actions if n in domain else ovl_actions).append(name)

    if rname:
        at = _simple(_first_target(rnode["always"]))
        ft = None
        for _, v in rnode["after"].items():
            ft = _simple(_first_target(v))
            break
        defs.append(f'RetryExhausted == st = "{rname}" /\\ rc >= MaxRetries /\\ st\' = "{at}" /\\ rc\' = rc')
        defs.append(f'RetryAgain == st = "{rname}" /\\ rc < MaxRetries /\\ st\' = "{ft}" /\\ rc\' = rc + 1')
        ovl_actions += ["RetryExhausted", "RetryAgain"]

    def setexpr(s):
        return "{" + ", ".join(f'"{x}"' for x in sorted(s)) + "}"

    lines = []
    lines.append(f"---- MODULE {mid} ----")
    lines.append(r"EXTENDS Naturals")
    lines.append("")
    lines.append(f"\\* Generated from {os.path.basename(path)} by tools/tla_gen.py. Control-flow model.")
    lines.append("CONSTANT MaxRetries")
    lines.append("VARIABLES st, rc")
    lines.append("vars == << st, rc >>")
    lines.append("")
    lines.append(f"States == {setexpr(set(names))}")
    lines.append(f"Domain == {setexpr(domain)}")
    lines.append(f"Overlay == {setexpr(overlay)}")
    lines.append("")
    lines.append("TypeOK == st \\in States /\\ rc \\in 0..MaxRetries")
    lines.append(f'Init == st = "{initial}" /\\ rc = 0')
    lines.append("")
    lines += comments
    lines.append("")
    lines += defs
    lines.append("")
    lines.append("DomainNext == " + (" \\/ ".join(dom_actions) if dom_actions else "FALSE"))
    lines.append("OverlayNext == " + (" \\/ ".join(ovl_actions) if ovl_actions else "FALSE"))
    lines.append("Next == DomainNext \\/ OverlayNext")
    lines.append("")
    lines.append("Spec == Init /\\ [][Next]_vars /\\ WF_vars(OverlayNext)")
    lines.append("")
    lines.append("Live_OverlayResolves == (st \\in Overlay) ~> (st \\in Domain)")
    lines.append("====")
    tla = "\n".join(lines)

    cfg = ("CONSTANT MaxRetries = 3\n"
           "SPECIFICATION Spec\n"
           "INVARIANT TypeOK\n"
           "PROPERTY Live_OverlayResolves\n")
    return mid, tla, cfg


def main():
    path = sys.argv[1]
    outdir = sys.argv[2] if len(sys.argv) > 2 else os.path.dirname(path)
    os.makedirs(outdir, exist_ok=True)
    mid, tla, cfg = generate(path)
    open(os.path.join(outdir, f"{mid}.tla"), "w").write(tla)
    open(os.path.join(outdir, f"{mid}.cfg"), "w").write(cfg)
    print(f"wrote {mid}.tla and {mid}.cfg to {outdir}")


if __name__ == "__main__":
    main()
