#!/usr/bin/env python3
"""compose_gen: generate a composition spec that checks cross-aggregate invariants
over the coordinator's REAL branching, validated against the coordinator machine.

The recursion-scaling step. Some invariants (no-ship-before-pay) span several
aggregates, so no single aggregate's model can state them. This tool:

  1. Loads the coordinator machine and VALIDATES the composition against it:
     each sequence entry names the coordinator state (`step:`) it belongs to,
     and the declared order must be exactly the machine's forward onDone chain.
     A composition that contradicts the coordinator is a hard error, never a
     green check.
  2. Models the full branching, not a happy-path script: every forward step can
     fail (first step fails clean, later steps enter compensation), compensation
     undoes obligations ONE AT A TIME in any order, and it can stall into the
     explicit FailedDirty residual with obligations still held.
  3. Auto-generates Inv_CleanCompensation (a clean Failed end state has undone
     every committed obligation) beside the user-declared ordering invariants,
     which are checked over ALL branches. Invariant expressions may reference
     `saga` (the coordinator state) and each aggregate variable.

What this does and does not prove: it proves the declared cross-aggregate
invariants over the composition whose step structure is validated against the
coordinator machine. That each aggregate actually conforms to its abstract
states here is the residual assumption, discharged per aggregate by its own
machine, oracle, and tests (and, where a semantics.yaml exists, its refinement).

Usage: compose_gen.py <composition.yaml> <coordinator.machine.json> [out-dir]
"""
import sys, os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import load_machine, walk_states, transitions_of  # noqa: E402

import yaml


def _t(s):
    return s[:1].upper() + s[1:]


def _simple(t):
    return t.lstrip("#").split(".")[-1] if t else None


def die(msg):
    sys.exit(f"compose_gen: VALIDATION FAILED: {msg}")


def forward_chain(machine):
    """The coordinator's forward path: initial, then each invoke onDone target,
    ending at a final state. Also returns the failure target of each step."""
    top = {n: node for p, n, node in walk_states(machine.get("states")) if "." not in p}
    chain, fails = [], {}
    cur = machine.get("initial")
    seen = set()
    while cur in top and top[cur].get("type") != "final":
        if cur in seen:
            die(f"forward chain loops at {cur!r}")
        seen.add(cur)
        node = top[cur]
        dones = {_simple(tr["target"]) for tr in transitions_of(node)
                 if tr["kind"] == "onDone" and tr["target"]}
        errs = {_simple(tr["target"]) for tr in transitions_of(node)
                if tr["kind"] in ("onError", "after") and tr["target"]}
        if len(dones) != 1:
            break  # not a forward step (e.g. Compensating routes elsewhere)
        chain.append(cur)
        fails[cur] = errs
        cur = next(iter(dones))
    return chain, fails, cur


def generate(comp, machine, machine_name):
    name = _t(comp["composition"])
    aggs = comp["aggregates"]
    seq = comp["sequence"]
    invs = comp.get("invariants", {})
    aggnames = list(aggs.keys())

    # ---- validate against the coordinator machine ----
    chain, fails, terminal = forward_chain(machine)
    declared = [s.get("step") for s in seq]
    if any(d is None for d in declared):
        die("every sequence entry needs step: <coordinator state>")
    if declared != chain:
        die(f"sequence steps {declared} do not match the coordinator's forward "
            f"chain {chain} (from {machine_name})")
    for i, s in enumerate(seq):
        st = s["step"]
        expected_fail = {"Failed"} if i == 0 else {"Compensating"}
        if fails.get(st) != expected_fail:
            die(f"step {st!r} failure paths in the machine go to {sorted(fails.get(st, ()))}, "
                f"expected {sorted(expected_fail)}")
        if s["aggregate"] not in aggs:
            die(f"step {st!r} names unknown aggregate {s['aggregate']!r}")
        if s["to"] not in aggs[s["aggregate"]]["states"]:
            die(f"step {st!r} commits {s['aggregate']} to unknown state {s['to']!r}")
        undo = s.get("undo")
        if i < len(seq) - 1 and not undo:
            die(f"step {st!r} needs an undo: (its compensating obligation); only the "
                f"completing step may omit it")
        if undo and undo.get("to") not in aggs[s["aggregate"]]["states"]:
            die(f"step {st!r} undo names unknown state {undo.get('to')!r}")

    top = {n: node for p, n, node in walk_states(machine.get("states")) if "." not in p}
    for needed in ("Compensating", "Completed", "Failed", "FailedDirty"):
        if needed not in top:
            die(f"coordinator has no {needed!r} state; the composition template "
                f"expects the saga pattern")

    saga_states = chain + ["Compensating", "Completed", "Failed", "FailedDirty"]
    obligations = [(s["aggregate"], s["to"], s["undo"]["to"])
                   for s in seq if s.get("undo")]

    # ---- emit ----
    L = []
    L.append("---- MODULE %s ----" % name)
    L.append("\\* GENERATED by tools/compose_gen.py from %s.composition.yaml," % comp["composition"])
    L.append("\\* VALIDATED against %s: the step order below IS the coordinator's" % machine_name)
    L.append("\\* forward onDone chain, and every failure route matches the machine.")
    L.append("\\* Models the full branching: step failures, per-obligation compensation")
    L.append("\\* in any order, and the FailedDirty stall with obligations still held.")
    L.append("\\* Residual assumption: each aggregate conforms to its abstract states,")
    L.append("\\* discharged per aggregate by its own machine, oracle, and tests.")
    L.append("")
    for a in aggnames:
        vals = ", ".join('"%s"' % s for s in aggs[a]["states"])
        L.append("%sStates == {%s}" % (_t(a), vals))
    L.append("SagaStates == {%s}" % ", ".join('"%s"' % s for s in saga_states))
    varlist = ", ".join(["saga"] + aggnames)
    L.append("VARIABLES %s" % varlist)
    L.append("vars == << %s >>" % varlist)
    L.append("")
    L.append("TypeOK == saga \\in SagaStates" +
             "".join(" /\\ %s \\in %sStates" % (a, _t(a)) for a in aggnames))
    L.append('Init == saga = "%s"' % chain[0] +
             "".join(' /\\ %s = "%s"' % (a, aggs[a]["initial"]) for a in aggnames))
    L.append("")
    acts = []

    def unch(exclude):
        keep = [a for a in aggnames if a not in exclude]
        return ("UNCHANGED << %s >>" % ", ".join(keep)) if keep else "TRUE"

    for i, s in enumerate(seq):
        st, a, to = s["step"], s["aggregate"], s["to"]
        nxt = chain[i + 1] if i + 1 < len(chain) else terminal
        L.append('Done_%s == saga = "%s" /\\ saga\' = "%s" /\\ %s\' = "%s" /\\ %s'
                 % (st, st, nxt, a, to, unch([a])))
        fail_to = "Failed" if i == 0 else "Compensating"
        L.append('Fail_%s == saga = "%s" /\\ saga\' = "%s" /\\ %s'
                 % (st, st, fail_to, unch([])))
        acts += ["Done_%s" % st, "Fail_%s" % st]
    for a, to, undo_to in obligations:
        L.append('Undo_%s == saga = "Compensating" /\\ %s = "%s" /\\ %s\' = "%s" '
                 '/\\ saga\' = saga /\\ %s' % (a, a, to, a, undo_to, unch([a])))
        acts.append("Undo_%s" % a)
    clean = " /\\ ".join('%s # "%s"' % (a, to) for a, to, _ in obligations)
    dirty = " \\/ ".join('%s = "%s"' % (a, to) for a, to, _ in obligations)
    L.append('CompensateDone == saga = "Compensating" /\\ (%s) /\\ saga\' = "Failed" /\\ %s'
             % (clean, unch([])))
    L.append('CompensateStall == saga = "Compensating" /\\ (%s) /\\ saga\' = "FailedDirty" /\\ %s'
             % (dirty, unch([])))
    acts += ["CompensateDone", "CompensateStall"]
    L.append('Done == saga \\in {"Completed", "Failed", "FailedDirty"} /\\ UNCHANGED vars')
    acts.append("Done")
    L.append("Next == %s" % " \\/ ".join(acts))
    L.append("Spec == Init /\\ [][Next]_vars /\\ WF_vars(Next)")
    L.append("")

    def cn(iname):
        return "Inv_" + "".join(w.capitalize() for w in iname.split("-"))

    L.append("\\* auto-generated: a clean Failed end has undone every committed obligation;")
    L.append("\\* only the explicit FailedDirty residual may still hold one")
    L.append('Inv_CleanCompensation == (saga = "Failed") => (%s)' % clean)
    for iname, expr in invs.items():
        L.append("%s == %s" % (cn(iname), expr))
    L.append('Live_Terminates == TRUE ~> (saga \\in {"Completed", "Failed", "FailedDirty"})')
    L.append("====")
    tla = "\n".join(L) + "\n"
    cfg = ("SPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_CleanCompensation\n"
           + "".join("INVARIANT %s\n" % cn(i) for i in invs)
           + "PROPERTY Live_Terminates\n")
    print(f"compose_gen: validated {name} against {machine_name}: "
          f"{len(chain)} forward steps, {len(obligations)} obligations, "
          f"{len(invs)} declared invariants")
    return name, tla, cfg


def main():
    with open(sys.argv[1], encoding="utf-8") as f:
        comp = yaml.safe_load(f)
    if len(sys.argv) < 3:
        sys.exit("usage: compose_gen.py <composition.yaml> <coordinator.machine.json> [out-dir]")
    machine, err = load_machine(sys.argv[2])
    if err:
        sys.exit(f"compose_gen: {err}")
    outdir = sys.argv[3] if len(sys.argv) > 3 else os.path.dirname(sys.argv[1])
    name, tla, cfg = generate(comp, machine, os.path.basename(sys.argv[2]))
    with open(os.path.join(outdir, name + ".tla"), "w", encoding="utf-8") as f:
        f.write(tla)
    with open(os.path.join(outdir, name + ".cfg"), "w", encoding="utf-8") as f:
        f.write(cfg)
    print("generated %s.tla + %s.cfg" % (name, name))


if __name__ == "__main__":
    main()
