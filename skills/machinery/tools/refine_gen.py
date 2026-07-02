#!/usr/bin/env python3
"""refine_gen: generate the data-refined model, the abstract contract, and the
refinement mapping for a machine, from a declarative semantics annotation, AFTER
reconciling that annotation against the machine.

This industrializes Parts 1 and 3 of the correctness ladder. The machine JSON
gives the control graph; the `<M>.semantics.yaml` sidecar gives the meaning the
graph omits (which stage order, which effects, which obligations). The sidecar is
a trust point, so it is NOT taken on faith: before emitting anything, this tool
verifies that the annotation and the machine agree (states match exactly, the
transition structure matches the declared pattern, the overlay has the expected
shape). A semantics file that drifts from the machine is a hard error, never a
green check. What remains assumed after reconciliation is printed and written
into the generated module header.

Emits, for `linear-lifecycle` (ordered open stages, win/lose terminals, reopen,
persist-retry overlay):
  <M>Data.tla / .cfg          domain invariants: stage-forward, atomicity, close-date
  <M>Contract.tla             the abstract resting/busy/atomic/terminates contract
  <M>Refinement.tla / .cfg    TLC-checked proof that <M>Data refines <M>Contract

and for `saga` (forward steps with compensating obligations):
  <M>Data.tla / .cfg          money and stock are never silently lost; compensation
                              is modeled PER OBLIGATION, so partial compensation
                              (refund succeeded, release failed) is representable
                              and FailedDirty is reachable in the model exactly
                              when an obligation may still be held.

Usage: refine_gen.py <machine.json> <semantics.yaml> [out-dir]
"""
import sys, os, json

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from machine_lint import walk_states, transitions_of, load_machine  # noqa: E402

import yaml


def _title(s):
    return s[:1].upper() + s[1:]


def _simple(t):
    return t.lstrip("#").split(".")[-1] if t else None


def die(msg):
    sys.exit(f"refine_gen: RECONCILIATION FAILED: {msg}")


def _top_states(machine):
    return {n: node for p, n, node in walk_states(machine.get("states")) if "." not in p}


def _on_targets(node, event):
    """Simple-name targets of an on-event's branches."""
    out = []
    for tr in transitions_of(node):
        if tr["kind"] == "on" and tr["event"] == event and tr["target"]:
            out.append(_simple(tr["target"]))
    return out


def _invoke_branch_targets(node, key):
    out = []
    for tr in transitions_of(node):
        if tr["kind"] == key and tr["target"]:
            out.append(_simple(tr["target"]))
    return out


# --------------------------- lifecycle pattern -----------------------------

def reconcile_lifecycle(machine, sem):
    """Verify the machine implements the linear-lifecycle pattern the semantics
    declare. Returns the set of domain states that can enter the overlay."""
    stages = sem["stages"]
    win, lose = sem["win_stage"], sem["lose_stage"]
    for key in ("advance_event", "win_event", "lose_event", "reopen_event"):
        if not sem.get(key):
            die(f"semantics must declare {key} (the Modelith action name) so the "
                f"machine's transition structure can be verified")
    adv, wev, lev, rev = (sem["advance_event"], sem["win_event"],
                          sem["lose_event"], sem["reopen_event"])
    top = _top_states(machine)
    domain_expected = set(stages) | {win, lose}
    domain_actual = {n for n in top if n[:1].isupper()}
    if domain_actual != domain_expected:
        die(f"domain states disagree: machine has {sorted(domain_actual)}, "
            f"semantics declare {sorted(domain_expected)}")
    if machine.get("initial") != stages[0]:
        die(f"machine initial is {machine.get('initial')!r}, semantics stage order "
            f"starts at {stages[0]!r}")
    for ov in ("persisting", "persistRetry", "rolledBack"):
        if ov not in top:
            die(f"overlay state {ov!r} missing from the machine")

    # advance edges: every non-final open stage advances into the overlay; the
    # last open stage does not advance; win/lose from every open stage; reopen
    # from terminals only
    for s in stages[:-1]:
        if "persisting" not in _on_targets(top[s], adv):
            die(f"stage {s!r} has no {adv!r} transition into persisting")
    if "persisting" in _on_targets(top[stages[-1]], adv):
        die(f"last open stage {stages[-1]!r} must not advance (win/lose only)")
    for s in stages:
        for ev, label in ((wev, "win"), (lev, "lose")):
            if "persisting" not in _on_targets(top[s], ev):
                die(f"open stage {s!r} has no {label} ({ev!r}) transition into persisting")
        if "persisting" in _on_targets(top[s], rev):
            die(f"open stage {s!r} must not reopen (terminals only)")
    for t in (win, lose):
        if "persisting" not in _on_targets(top[t], rev):
            die(f"terminal {t!r} has no reopen ({rev!r}) transition into persisting")
        for ev in (adv, wev, lev):
            if "persisting" in _on_targets(top[t], ev):
                die(f"terminal {t!r} must reject {ev!r}, not persist it")

    # overlay shape
    ondone = set(_invoke_branch_targets(top["persisting"], "onDone"))
    expected_commits = set(stages[1:]) | {win, lose}
    if not expected_commits <= ondone:
        die(f"persisting onDone commits to {sorted(ondone)}; expected at least "
            f"{sorted(expected_commits)} (every advance/win/lose target)")
    if not ondone <= expected_commits | {"rolledBack"}:
        die(f"persisting onDone reaches unexpected states {sorted(ondone - expected_commits - {'rolledBack'})}")
    onerror = set(_invoke_branch_targets(top["persisting"], "onError"))
    if not onerror <= {"persistRetry", "rolledBack"}:
        die(f"persisting onError reaches unexpected states {sorted(onerror)}")
    retry_always = set(_invoke_branch_targets(top["persistRetry"], "always")) | \
        set(t for t in [_simple(b.get("target")) for b in
            (top["persistRetry"].get("always") or []) if isinstance(b, dict)] if t)
    if retry_always != {"rolledBack"}:
        die(f"persistRetry always must go to rolledBack (found {sorted(retry_always)})")
    rb_targets = {_simple(b.get("target")) for b in (top["rolledBack"].get("always") or [])
                  if isinstance(b, dict)}
    # domain states that can actually enter the overlay
    enters = {s for s in domain_actual if "persisting" in
              {_simple(tr["target"]) for tr in transitions_of(top[s]) if tr["target"]}}
    if rb_targets != enters:
        die(f"rolledBack routes to {sorted(rb_targets)} but the overlay is entered "
            f"from {sorted(enters)}; the rollback routing is incomplete or stale")
    if sem["close_date_on"] not in domain_expected:
        die(f"close_date_on {sem['close_date_on']!r} is not a domain state")
    return enters


def emit_lifecycle(machine, sem, source_names):
    mid = _title(sem["machine"])
    reconciled_from = reconcile_lifecycle(machine, sem)
    stages = sem["stages"]
    win, lose = sem["win_stage"], sem["lose_stage"]
    reopen_to = sem["reopen_to"]
    close_on = sem["close_date_on"]
    maxr = sem["max_retries"]
    initial = stages[0]

    terminal = [win, lose]
    domain = stages + terminal
    advanceable = stages[:-1]
    rank = {s: i for i, s in enumerate(stages)}
    rank[win] = rank[lose] = len(stages)
    nxt = {stages[i]: stages[i + 1] for i in range(len(stages) - 1)}

    def q(xs):
        return "{" + ", ".join(f'"{x}"' for x in xs) + "}"

    rankf = "[" + ", ".join(f"{s} |-> {rank[s]}" for s in domain) + "]"
    nextf = "[" + ", ".join(f'{k} |-> "{v}"' for k, v in nxt.items()) + "]"

    header = f"""\\* GENERATED by tools/refine_gen.py from {source_names[0]} + {source_names[1]}.
\\* Data-refined model: proves the real domain invariants, not just control flow.
\\*
\\* RECONCILED against the machine before emission: domain states, initial, the
\\* advance/win/lose/reopen transition structure, the overlay shape, and the
\\* rollback routing all match the machine JSON; a drifted semantics file is a
\\* hard generation error.
\\* STILL ASSUMED (outside the machine JSON, carried by the named-unit contracts
\\* and the implementation tests): the pending/prior context updates the actions
\\* perform, the retry bound MaxRetries = {maxr}, and single-instance execution."""

    data = f"""---- MODULE {mid}Data ----
{header}
EXTENDS Naturals

CONSTANT MaxRetries

Open == {q(stages)}
Terminal == {q(terminal)}
Domain == Open \\cup Terminal
Overlay == {{"persisting", "persistRetry", "rolledBack"}}
None == "none"
Rank == {rankf}
NextStage == {nextf}

VARIABLES st, rc, stage, pending, prior, closeSet
vars == << st, rc, stage, pending, prior, closeSet >>

TypeOK ==
  /\\ st \\in (Domain \\cup Overlay)
  /\\ rc \\in 0..MaxRetries
  /\\ stage \\in Domain
  /\\ pending \\in (Domain \\cup {{None}})
  /\\ prior \\in (Domain \\cup {{None}})
  /\\ closeSet \\in BOOLEAN

Init ==
  /\\ st = "{initial}" /\\ stage = "{initial}"
  /\\ rc = 0 /\\ pending = None /\\ prior = None /\\ closeSet = FALSE

StartAdvance ==
  /\\ st \\in {q(advanceable)}
  /\\ st' = "persisting" /\\ pending' = NextStage[st] /\\ prior' = st
  /\\ rc' = 0 /\\ stage' = stage /\\ closeSet' = closeSet

StartWin ==
  /\\ st \\in Open
  /\\ st' = "persisting" /\\ pending' = "{win}" /\\ prior' = st
  /\\ rc' = 0 /\\ stage' = stage /\\ closeSet' = closeSet

StartLose ==
  /\\ st \\in Open
  /\\ st' = "persisting" /\\ pending' = "{lose}" /\\ prior' = st
  /\\ rc' = 0 /\\ stage' = stage /\\ closeSet' = closeSet

StartReopen ==
  /\\ st \\in Terminal
  /\\ st' = "persisting" /\\ pending' = "{reopen_to}" /\\ prior' = st
  /\\ rc' = 0 /\\ stage' = stage /\\ closeSet' = closeSet

SaveDone ==
  /\\ st = "persisting"
  /\\ st' = pending /\\ stage' = pending
  /\\ closeSet' = (closeSet \\/ (pending = "{close_on}"))
  /\\ pending' = None /\\ prior' = None /\\ rc' = 0

SaveLocked ==
  /\\ st = "persisting" /\\ st' = "persistRetry"
  /\\ UNCHANGED << rc, stage, pending, prior, closeSet >>

SaveFail ==
  /\\ st = "persisting" /\\ st' = "rolledBack"
  /\\ UNCHANGED << rc, stage, pending, prior, closeSet >>

RetryExhausted ==
  /\\ st = "persistRetry" /\\ rc >= MaxRetries /\\ st' = "rolledBack"
  /\\ UNCHANGED << rc, stage, pending, prior, closeSet >>

RetryAgain ==
  /\\ st = "persistRetry" /\\ rc < MaxRetries /\\ st' = "persisting" /\\ rc' = rc + 1
  /\\ UNCHANGED << stage, pending, prior, closeSet >>

RolledBack ==
  /\\ st = "rolledBack"
  /\\ st' = prior /\\ stage' = prior
  /\\ pending' = None /\\ prior' = None /\\ rc' = 0 /\\ closeSet' = closeSet

Domain_Next == StartAdvance \\/ StartWin \\/ StartLose \\/ StartReopen
Overlay_Next == SaveDone \\/ SaveLocked \\/ SaveFail \\/ RetryExhausted \\/ RetryAgain \\/ RolledBack
Next == Domain_Next \\/ Overlay_Next

Spec == Init /\\ [][Next]_vars /\\ WF_vars(Overlay_Next)

Inv_StageValid == stage \\in Domain
Inv_Atomic == (st \\in Overlay) => (stage = prior)
Inv_DomainConsistent == (st \\in Domain) => (st = stage /\\ pending = None /\\ prior = None)
Inv_CloseDate == (stage = "{close_on}") => closeSet

StageForward ==
  [][ (stage' # stage) =>
        \\/ Rank[stage'] > Rank[stage]
        \\/ (stage \\in Terminal /\\ stage' = "{reopen_to}") ]_stage

Live_OverlayResolves == (st \\in Overlay) ~> (st \\in Domain)
====
"""

    data_cfg = ("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\n"
                "INVARIANT TypeOK\nINVARIANT Inv_StageValid\nINVARIANT Inv_Atomic\n"
                "INVARIANT Inv_DomainConsistent\nINVARIANT Inv_CloseDate\n"
                "PROPERTY StageForward\nPROPERTY Live_OverlayResolves\n") % maxr

    contract = f"""---- MODULE {mid}Contract ----
\\* GENERATED. The abstract contract the big picture assumes of the {sem['machine']}
\\* aggregate: resting or busy, atomic while busy, and every busy period terminates.
VARIABLES phase, kind
cvars == << phase, kind >>

Phases == {{"resting", "busy"}}
Kinds == {{"open", "terminal"}}

CTypeOK == phase \\in Phases /\\ kind \\in Kinds
CInit == phase = "resting" /\\ kind = "open"

Begin == phase = "resting" /\\ phase' = "busy" /\\ kind' = kind
Finish == phase = "busy" /\\ phase' = "resting" /\\ kind' \\in Kinds
Churn == phase = "busy" /\\ phase' = "busy" /\\ kind' = kind
RestStutter == phase = "resting" /\\ UNCHANGED cvars

CNext == Begin \\/ Finish \\/ Churn \\/ RestStutter
CSpec == CInit /\\ [][CNext]_cvars /\\ WF_cvars(Finish)
CTermination == (phase = "busy") ~> (phase = "resting")
====
"""

    refinement = f"""---- MODULE {mid}Refinement ----
\\* GENERATED. Proof that {mid}Data refines {mid}Contract under a refinement mapping.
EXTENDS {mid}Data

phaseBar == IF st \\in Domain THEN "resting" ELSE "busy"
kindBar == IF stage \\in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE {mid}Contract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
"""

    ref_cfg = ("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\n"
               "INVARIANT RefTypeOK\nPROPERTY RefSpec\nPROPERTY RefTermination\n") % maxr

    print(f"refine_gen: reconciled {mid} against the machine: "
          f"{len(stages) + 2} domain states, overlay entered from {len(reconciled_from)} states")
    return mid, {f"{mid}Data.tla": data, f"{mid}Data.cfg": data_cfg,
                 f"{mid}Contract.tla": contract,
                 f"{mid}Refinement.tla": refinement, f"{mid}Refinement.cfg": ref_cfg}


# ------------------------------ saga pattern -------------------------------

def reconcile_saga(machine, sem):
    """Verify the machine implements the saga pattern the semantics declare."""
    states = sem["states"]
    obl = sem["obligations"]
    top = _top_states(machine)
    expected = set(states) | {"Compensating", "compensateRetry",
                              "Completed", "Failed", "FailedDirty"}
    if set(top) != expected:
        die(f"saga states disagree: machine has {sorted(top)}, semantics imply "
            f"{sorted(expected)}")
    if machine.get("initial") != states[0]:
        die(f"machine initial is {machine.get('initial')!r}, first forward step is "
            f"{states[0]!r}")
    for i, s in enumerate(states):
        nxt = states[i + 1] if i + 1 < len(states) else "Completed"
        ondone = set(_invoke_branch_targets(top[s], "onDone"))
        if ondone != {nxt}:
            die(f"forward step {s!r} onDone goes to {sorted(ondone)}, expected {nxt!r}")
        fail_to = "Failed" if i == 0 else "Compensating"
        onerr = set(_invoke_branch_targets(top[s], "onError"))
        after = {_simple(tr["target"]) for tr in transitions_of(top[s])
                 if tr["kind"] == "after" and tr["target"]}
        if onerr != {fail_to} or after != {fail_to}:
            die(f"forward step {s!r} failure paths go to onError={sorted(onerr)}, "
                f"after={sorted(after)}; expected {fail_to!r} (first step fails clean, "
                f"later steps compensate)")
    comp = top["Compensating"]
    if set(_invoke_branch_targets(comp, "onDone")) != {"Failed"}:
        die("Compensating onDone must reach Failed (compensation complete)")
    if set(_invoke_branch_targets(comp, "onError")) != {"compensateRetry"}:
        die("Compensating onError must reach compensateRetry")
    cr = top["compensateRetry"]
    cr_always = {_simple(b.get("target")) for b in (cr.get("always") or [])
                 if isinstance(b, dict)}
    cr_after = {_simple(tr["target"]) for tr in transitions_of(cr)
                if tr["kind"] == "after" and tr["target"]}
    if cr_always != {"FailedDirty"} or cr_after != {"Compensating"}:
        die("compensateRetry must exhaust to FailedDirty and back off to Compensating")
    for f in ("Completed", "Failed", "FailedDirty"):
        if top[f].get("type") != "final":
            die(f"{f} must be a final state")
    for s in states[:-1]:
        if not (obl.get(s, {}).get("sets") and obl.get(s, {}).get("undo")):
            die(f"forward step {s!r} must declare sets: and undo: (its compensating "
                f"obligation); only the completing step may omit undo")
    if not obl.get(states[-1], {}).get("sets"):
        die(f"completing step {states[-1]!r} must declare sets:")
    unknown = set(obl) - set(states)
    if unknown:
        die(f"obligations declared for unknown steps: {sorted(unknown)}")


def emit_saga(machine, sem, source_names):
    """Saga: forward steps commit side effects with compensating obligations.
    Compensation is modeled PER OBLIGATION: each undo is its own step, so a
    partially compensated saga (refund done, release not) is a real state, the
    retry resumes idempotently from where it stopped, and FailedDirty is
    reachable exactly when an obligation may still be held."""
    mid = _title(sem["machine"])
    reconcile_saga(machine, sem)
    states = sem["states"]
    obl = sem["obligations"]
    maxr = sem["max_retries"]
    initial = states[0]

    flags = []
    for s in states:
        for k in ("sets", "undo"):
            v = obl.get(s, {}).get(k)
            if v and v not in flags:
                flags.append(v)
    obligations = [(obl[s]["sets"], obl[s]["undo"]) for s in states
                   if obl.get(s, {}).get("sets") and obl.get(s, {}).get("undo")]
    varlist = ", ".join(["st", "rc"] + flags)

    def unch(exclude):
        keep = [v for v in (["rc"] + flags) if v not in exclude]
        return "<< " + ", ".join(keep) + " >>"

    L = []
    L.append(f"---- MODULE {mid}Data ----")
    L.append(f"\\* GENERATED by tools/refine_gen.py (saga pattern) from {source_names[0]} + {source_names[1]}.")
    L.append("\\* Proves money and stock are never silently lost: a terminal saga has undone")
    L.append("\\* every obligation it committed, or ends FailedDirty as an explicit residual.")
    L.append("\\*")
    L.append("\\* RECONCILED against the machine before emission: step order, failure routing,")
    L.append("\\* the compensation loop, and the final states all match the machine JSON.")
    L.append("\\* Compensation here is PER OBLIGATION (each undo its own step), refining the")
    L.append("\\* machine's single idempotent compensate invoke, so partial compensation is")
    L.append("\\* representable. STILL ASSUMED: the obligation flags mirror what the real")
    L.append(f"\\* actors commit and undo, the retry bound MaxRetries = {maxr}, single instance.")
    L.append("EXTENDS Naturals")
    L.append("")
    L.append("CONSTANT MaxRetries")
    L.append("Steps == {" + ", ".join(f'"{s}"' for s in states) + ', "Compensating", "compensateRetry"}')
    L.append('Final == {"Completed", "Failed", "FailedDirty"}')
    L.append(f"VARIABLES {varlist}")
    L.append(f"vars == << {varlist} >>")
    L.append("")
    L.append("TypeOK == st \\in (Steps \\cup Final) /\\ rc \\in 0..MaxRetries"
             + "".join(f" /\\ {f} \\in BOOLEAN" for f in flags))
    L.append(f'Init == st = "{initial}" /\\ rc = 0' + "".join(f" /\\ {f} = FALSE" for f in flags))
    L.append("")
    overlay = []
    for i, s in enumerate(states):
        nxt = states[i + 1] if i + 1 < len(states) else "Completed"
        sets = obl.get(s, {}).get("sets")
        eff = f" /\\ {sets}' = TRUE" if sets else ""
        L.append(f'Done_{s} == st = "{s}" /\\ st\' = "{nxt}"{eff} /\\ UNCHANGED {unch([sets] if sets else [])}')
        ft = "Failed" if i == 0 else "Compensating"
        L.append(f'Fail_{s} == st = "{s}" /\\ st\' = "{ft}" /\\ UNCHANGED {unch([])}')
        overlay += [f"Done_{s}", f"Fail_{s}"]
    # per-obligation compensation: one undo at a time, in any order
    open_obl = " \\/ ".join(f"({sv} /\\ ~{u})" for sv, u in obligations)
    all_clean = " /\\ ".join(f"({sv} => {u})" for sv, u in obligations)
    for sv, u in obligations:
        L.append(f'Undo_{u} == st = "Compensating" /\\ {sv} /\\ ~{u} /\\ {u}\' = TRUE '
                 f'/\\ st\' = st /\\ UNCHANGED {unch([u])}')
        overlay.append(f"Undo_{u}")
    L.append(f'CompensateDone == st = "Compensating" /\\ ({all_clean}) /\\ st\' = "Failed" /\\ UNCHANGED {unch([])}')
    L.append(f'CompensateErr == st = "Compensating" /\\ ({open_obl}) /\\ st\' = "compensateRetry" /\\ UNCHANGED {unch([])}')
    L.append(f'RetryExhausted == st = "compensateRetry" /\\ rc >= MaxRetries /\\ st\' = "FailedDirty" /\\ UNCHANGED {unch([])}')
    L.append(f'RetryAgain == st = "compensateRetry" /\\ rc < MaxRetries /\\ st\' = "Compensating" /\\ rc\' = rc + 1 /\\ UNCHANGED {unch(["rc"])}')
    overlay += ["CompensateDone", "CompensateErr", "RetryExhausted", "RetryAgain"]
    L.append("")
    L.append("OverlayNext == " + " \\/ ".join(overlay))
    L.append("Terminated == st \\in Final /\\ UNCHANGED vars")
    L.append("Next == OverlayNext \\/ Terminated")
    L.append("Spec == Init /\\ [][Next]_vars /\\ WF_vars(OverlayNext)")
    L.append("")
    nsl = " /\\ ".join(f'(({sv} /\\ st # "Completed") => ({u} \\/ st = "FailedDirty"))' for sv, u in obligations)
    L.append(f"Inv_NoSilentLoss == (st \\in Final) => ({nsl})")
    L.append(f'Inv_CleanCompensation == (st = "Failed") => ({all_clean})')
    L.append("Live_Terminates == (st \\notin Final) ~> (st \\in Final)")
    L.append("====")
    tla = "\n".join(L) + "\n"
    cfg = (f"CONSTANT MaxRetries = {maxr}\nSPECIFICATION Spec\nINVARIANT TypeOK\n"
           "INVARIANT Inv_NoSilentLoss\nINVARIANT Inv_CleanCompensation\n"
           "PROPERTY Live_Terminates\n")
    print(f"refine_gen: reconciled {mid} against the machine: {len(states)} forward "
          f"steps, {len(obligations)} compensating obligations")
    return mid, {f"{mid}Data.tla": tla, f"{mid}Data.cfg": cfg}


def main():
    machine, err = load_machine(sys.argv[1])
    if err:
        sys.exit(f"refine_gen: {err}")
    with open(sys.argv[2], encoding="utf-8") as f:
        sem = yaml.safe_load(f)
    outdir = sys.argv[3] if len(sys.argv) > 3 else os.path.dirname(sys.argv[2])
    names = (os.path.basename(sys.argv[1]), os.path.basename(sys.argv[2]))
    pat = sem.get("pattern")
    if pat == "linear-lifecycle":
        mid, files = emit_lifecycle(machine, sem, names)
    elif pat == "saga":
        mid, files = emit_saga(machine, sem, names)
    else:
        sys.exit(f"refine_gen: unsupported pattern {pat!r} (linear-lifecycle, saga)")
    for name, body in files.items():
        with open(os.path.join(outdir, name), "w", encoding="utf-8") as f:
            f.write(body)
    print(f"generated {len(files)} files for {mid} ({pat})")


if __name__ == "__main__":
    main()
