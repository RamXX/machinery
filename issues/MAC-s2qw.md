---
id: MAC-s2qw
title: "Hook-state store: sanctioned agent-seat handoff, truthful refusal message, doctor sees the binding"
status: open
priority: 1
type: bug
labels: [hook, governance, doctor, h2, from-next]
created_at: 2026-09-24T21:32:49Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:49Z
content_hash: "sha256:44780adfe837eac2cdc1aa2d3caafd9c0f9eb585feb5114719af57aeb44dfed6"
related: [MAC-bmlh]
---

## Description
Hook-state store: sanctioned agent-seat handoff, a truthful refusal, and a doctor that sees the binding

Problem (H2, 2026-09-19, installed v0.8.0): the H2 conductor seat rotated from a Claude Code agent (weekly quota exhausted) to an OpenCode agent on the same machine, same checkout, same binary. Every governed tool call from the new seat was denied: "machinery governance cannot access its durable project-obligation store: durable hook state directory ... changed native identity; refusing to accept a replacement store". The store's `.store-identity` recorded `directory unix:1000011:1cf2ac2e`; `ls -di` on the live directory printed 485665838, that exact inode, so the store was the original. The same binary and payload against a different root created a fresh store and allowed the call. `machinery doctor` reported the store healthy (476 entries), and the plugin's failure guidance says "run machinery doctor".

Defects:
1. The refusal names the wrong cause. It frames an intact store as a rejected replacement; the real mismatch is what the generation binds (a caller-shaped identity of the seat that created the store). Code: internal/hook/hook.go:2270 and :2273 emit "refusing to accept a replacement store" for both a native-identity mismatch and an initialization-marker mismatch.
2. Doctor cannot see this failure class: integrity and entry counts pass while the hook denies every call.
3. No sanctioned seat transition. The installer supports three native targets and the skill states one contract across hosts, yet one seat handing a project to another on the same root leaves governance hard-down. The only remedy is hand-moving the private state directory (the action the error says machinery refuses), which silently discards the pending-obligation ledger.
4. The identity model is undocumented: nothing installed says what the store binds beyond the root path hash (directory device:inode, a generation over caller inputs).

Addendum findings from the same incident:
(a) The darwin native witness binds OS-volatile components: device id and inode generation (`st_gen`) move on macOS under an OS update or restore with no tampering. That is what fired here. A witness that changes when nothing was touched is not a tamper witness.
(b) The loss sentinel defeats the quarantine remedy. Quarantining the store and letting the first call create a fresh one still refuses with "missing after prior initialization" because `~/.machinery-hook-state-<key>.initialized` (hook.go:2065) outlives the store by design. The real clearing sequence (quarantine the store AND remove the sentinel) was found only from source.

Proposed fix:
- Name the mismatched component in the error (for example: store bound to native host X, this caller is Y); reserve "refusing to accept a replacement store" for a real device:inode mismatch.
- Drop OS-volatile fields (`st_gen`, device id where it moves across OS updates) from the darwin witness, or classify their change as a non-tamper event with its own message.
- A first-class transition, for example `machinery hook-state adopt --root <root>`: verify the existing store's integrity, re-bind it to the current caller, journal the handoff, and report pending obligations in the old ledger (touched files not yet green) instead of dropping them. The same command (or a documented sibling) handles the sentinel so a quarantine remedy is a supported path.
- Doctor exercises the same binding check the hook enforces and, on failure, names the transition command instead of reporting ok.
- Document the identity model (what is bound, why, which transitions are sanctioned, the sentinel) in the installed skill or CLI reference.

Acceptance criteria:
1. A synthetic one-root, two-caller reproduction (two native hosts, or a native host plus the bare hook protocol from a shell): the second caller's first call is denied with a message naming the mismatched component, not "replacement store".
2. `machinery doctor` in that state exits nonzero, reports the binding failure, and names the transition command.
3. The transition re-binds the store, journals the handoff, preserves and reports pending ledger obligations, and the second caller proceeds with no manual state-directory surgery.
4. A genuinely replaced store directory (new inode) still refuses with the replacement message.
5. A darwin witness whose `st_gen` or device id changed with the inode unchanged is not reported as tampering (unit test over the witness comparison).
6. The documented clearing path covers the loss sentinel; a quarantine followed by the documented steps yields a working store.
7. Existing single-seat flows are unchanged and the fail-closed direction is preserved (existing hook tests stay green).
8. The installed skill or CLI reference documents the identity model and the sanctioned transitions.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Related: [[MAC-bmlh]]

## Comments
