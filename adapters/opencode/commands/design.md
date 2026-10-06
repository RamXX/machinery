---
description: Start or resume a gated machinery design run
---

Load the `machinery` skill and follow it as the conductor. Speak to the user
in the skill's dialog register: plain step names (the domain model, the
architecture, the behavior model, the build plan), findings translated to
their meaning, no gate ids, phase numbers, or CLI invocations unless the
user uses them first; internally the skill's four-phase workflow and its
gates apply unchanged. Resume from `design/STATE.md` when present. Treat the
request as greenfield, brownfield, hybrid, or rebuild according to the
skill's mode rules. Ask what an adversary would do as well as what users do:
anything that verifies, authenticates, signs, hashes, approves, admits input,
or gates a decision follows the skill's threat-modeling reference, and
security-relevant work is handed over as a threat table plus paired
accept/refuse criteria. If the request is empty, open by asking in plain
language what we are building, who uses it, and what it should be written
in.

Request: $ARGUMENTS
