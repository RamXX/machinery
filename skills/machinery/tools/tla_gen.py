#!/usr/bin/env python3
"""tla_gen: translate a machinery machine (the IR) into a TLA+ control-flow model.

Rung 3 of the correctness ladder. Emits a TLA+ module plus a TLC config from a
machine JSON so the machine's control graph is model-checked exhaustively rather
than reviewed by eye. Every generated module carries an ASSUMPTIONS block stating
exactly what the abstraction erases, so the green check cannot read as more than
it is.

The model:
  - `st` ranges over the top-level machine states.
  - Every retry-shaped state (a state with a guarded `always` and an `after`)
    gets its OWN bounded counter, so machines with several retry loops (a
    gateway retry plus a persist retry) are modeled faithfully; the old
    first-match-only behavior mismodeled the second loop as unbounded.
  - Guards are over-approximated (every branch of a guarded list is possible).
    SOUND for safety: properties that hold over the larger behavior set hold
    for the guarded subset. For liveness this is CONDITIONAL on the guard lists
    being exhaustive, which machine_lint discharges: every fully guarded
    `always` list must carry an unguarded fallback or an _exhaustive note.

Hard errors instead of silent narrowing: nested/compound states, parallel
machines, and multiple `after` entries on a retry state are rejected with an
instruction, never quietly dropped.

Generated properties:
  TypeOK               (structural lock) states valid, counters bounded.
  Live_OverlayResolves (the substantive one) from any transient overlay state
                       the machine always eventually returns to a resting
                       domain state: no unbounded retry, nothing half-done.
TLC also checks deadlock-freedom by default. TypeOK and deadlock-freedom are
largely guaranteed by construction for generator output; they are regression
locks against hand-edited specs, not discoveries.

Usage: tla_gen.py <machine.json> [out-dir]
"""
import sys, os, json

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of, load_machine, invokes_of  # noqa: E402


def classify(states):
    domain, overlay = set(), set()
    for _, n, node in states:
        if node.get("on") or node.get("type") == "final":
            domain.add(n)
        else:
            overlay.add(n)
    return domain, overlay


def retry_states(states):
    """Every state with a guarded always plus an after: a bounded retry loop."""
    out = []
    for _, n, node in states:
        if node.get("always") and node.get("after"):
            branches = node["always"] if isinstance(node["always"], list) else [node["always"]]
            if all(isinstance(b, dict) and b.get("guard") for b in branches):
                out.append((n, node))
    return out


def _targets_of(x, what, mid):
    """All targets of a transition value. Guarded routing (e.g. a retry that
    resumes to one of several phases) becomes a nondeterministic choice, which
    is the guard-erasure abstraction applied consistently."""
    items = x if isinstance(x, list) else [x]
    targets = []
    for it in items:
        if isinstance(it, dict) and it.get("target"):
            targets.append(it["target"])
        elif isinstance(it, str):
            targets.append(it)
    if not targets:
        sys.exit(f"tla_gen: {mid}: {what} has no target; the retry template needs one")
    return targets


def _simple(t):
    return t.lstrip("#").split(".")[-1] if t else None


def generate(path):
    m, err = load_machine(path)
    if err:
        sys.exit(f"tla_gen: {err}")
    mid = m.get("id", "machine")
    mid = mid[:1].upper() + mid[1:]

    all_states = walk_states(m.get("states"))
    nested = [p for p, _, _ in all_states if "." in p]
    if nested:
        sys.exit(f"tla_gen: {mid}: nested states are not supported at rung 3 "
                 f"({', '.join(sorted(nested))}); flatten the machine or extend the generator")
    for _, n, node in all_states:
        if node.get("type") not in (None, "atomic", "compound", "final"):
            sys.exit(f"tla_gen: {mid}: unsupported state type {node.get('type')!r} in {n}")

    states = all_states
    names = [n for _, n, _ in states]
    domain, overlay = classify(states)
    retries = retry_states(states)
    rc_of = {n: f"rc{i + 1}" for i, (n, _) in enumerate(retries)}
    counters = list(rc_of.values())
    initial = m.get("initial")
    final = sorted(n for _, n, node in states if node.get("type") == "final")

    exhaustive_notes = []
    for _, n, node in states:
        note = node.get("_exhaustive")
        if isinstance(note, str) and note.strip():
            exhaustive_notes.append((n, note.strip()))

    def counter_updates(src, tgt):
        """Counter assignments for a transition src -> tgt. A transition out of
        a domain state starts a fresh operation: every counter resets. Other
        transitions leave counters unchanged (retry steps override their own)."""
        ups = {}
        for rn, var in rc_of.items():
            if src in domain:
                ups[var] = "0"
            elif tgt in domain:
                ups[var] = "0"
            else:
                ups[var] = var
        return ups

    dom_actions, ovl_actions, defs, comments = [], [], [], []
    idx = 0
    for _, n, node in states:
        if n in rc_of:
            continue  # handled by the retry template below
        for tr in transitions_of(node):
            idx += 1
            tgt = _simple(tr["target"]) or n
            name = f"T{idx}"
            ups = counter_updates(n, tgt)
            parts = [f'st = "{n}"', f'st\' = "{tgt}"']
            parts += [f"{var}' = {val}" for var, val in ups.items()]
            defs.append(f"{name} == " + " /\\ ".join(parts))
            trig = f'{tr["kind"]}:{tr["event"]}' if tr.get("event") else tr["kind"]
            comments.append(f"  \\* {name}: {n} -{trig}-> {tgt}")
            (dom_actions if n in domain else ovl_actions).append(name)

    def st_step(targets):
        ts = sorted({_simple(t) for t in targets})
        if len(ts) == 1:
            return f'st\' = "{ts[0]}"'
        return "st' \\in {" + ", ".join(f'"{t}"' for t in ts) + "}"

    for rn, rnode in retries:
        var = rc_of[rn]
        a_step = st_step(_targets_of(rnode["always"], f"retry state {rn} always", mid))
        if len(rnode["after"]) != 1:
            sys.exit(f"tla_gen: {mid}: retry state {rn} has {len(rnode['after'])} after "
                     f"entries; the retry template needs exactly one")
        f_step = st_step(_targets_of(next(iter(rnode["after"].values())),
                                     f"retry state {rn} after", mid))
        others = [v for v in counters if v != var]
        unch = (" /\\ " + " /\\ ".join(f"{v}' = {v}" for v in others)) if others else ""
        defs.append(f'RetryExhausted_{rn} == st = "{rn}" /\\ {var} >= MaxRetries '
                    f'/\\ {a_step} /\\ {var}\' = {var}{unch}')
        defs.append(f'RetryAgain_{rn} == st = "{rn}" /\\ {var} < MaxRetries '
                    f'/\\ {f_step} /\\ {var}\' = {var} + 1{unch}')
        ovl_actions += [f"RetryExhausted_{rn}", f"RetryAgain_{rn}"]

    if final:
        defs.append("Terminated == st \\in Final /\\ UNCHANGED vars")

    def setexpr(s):
        return "{" + ", ".join(f'"{x}"' for x in sorted(s)) + "}"

    varlist = ", ".join(["st"] + counters)
    lines = []
    lines.append(f"---- MODULE {mid} ----")
    lines.append(r"EXTENDS Naturals")
    lines.append("")
    lines.append(f"\\* Generated from {os.path.basename(path)} by tools/tla_gen.py. Control-flow model.")
    lines.append("\\*")
    lines.append("\\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):")
    lines.append("\\*   1. Guards are erased to nondeterminism: sound for safety; for liveness the")
    lines.append("\\*      guard lists must be exhaustive. machine_lint enforces an unguarded")
    lines.append("\\*      fallback or an _exhaustive note on every fully guarded always-list.")
    for n, note in exhaustive_notes:
        lines.append(f"\\*      - {n}: {note}")
    lines.append("\\*   2. Every invoke resolves exactly once (onDone or onError; no lost or")
    lines.append("\\*      duplicated completion) and every after timer eventually fires.")
    lines.append("\\*   3. Single machine instance; no interleaving with other instances or")
    lines.append("\\*      machines, no message loss/duplication/reordering between machines.")
    lines.append("\\*   4. Context data, event payloads, action effects, and real time (the")
    lines.append("\\*      _delays values) are not modeled at this rung; the data-refined rung")
    lines.append("\\*      (refine_gen) and the implementation tests carry those.")
    lines.append("CONSTANT MaxRetries")
    lines.append(f"VARIABLES {varlist}")
    lines.append(f"vars == << {varlist} >>")
    lines.append("")
    lines.append(f"States == {setexpr(set(names))}")
    lines.append(f"Domain == {setexpr(domain)}")
    lines.append(f"Overlay == {setexpr(overlay)}")
    if final:
        lines.append(f"Final == {setexpr(set(final))}")
    lines.append("")
    tycounts = " /\\ ".join([f"{v} \\in 0..MaxRetries" for v in counters]) or "TRUE"
    lines.append(f"TypeOK == st \\in States /\\ {tycounts}")
    init_counts = "".join(f" /\\ {v} = 0" for v in counters)
    lines.append(f'Init == st = "{initial}"{init_counts}')
    lines.append("")
    lines += comments
    lines.append("")
    lines += defs
    lines.append("")
    lines.append("DomainNext == " + (" \\/ ".join(dom_actions) if dom_actions else "FALSE"))
    lines.append("OverlayNext == " + (" \\/ ".join(ovl_actions) if ovl_actions else "FALSE"))
    lines.append("Next == DomainNext \\/ OverlayNext" + (" \\/ Terminated" if final else ""))
    lines.append("")
    lines.append("Spec == Init /\\ [][Next]_vars /\\ WF_vars(OverlayNext)")
    lines.append("")
    lines.append("Live_OverlayResolves == (st \\in Overlay) ~> (st \\in Domain)")
    lines.append("====")
    tla = "\n".join(lines) + "\n"

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
    with open(os.path.join(outdir, f"{mid}.tla"), "w", encoding="utf-8") as f:
        f.write(tla)
    with open(os.path.join(outdir, f"{mid}.cfg"), "w", encoding="utf-8") as f:
        f.write(cfg)
    print(f"wrote {mid}.tla and {mid}.cfg to {outdir}")


if __name__ == "__main__":
    main()
