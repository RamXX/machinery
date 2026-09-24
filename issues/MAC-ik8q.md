---
id: MAC-ik8q
title: "Governed early executable experiments for architecture assumptions"
status: open
priority: 3
type: feature
labels: [experiments, evidence, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:433527a79149cd8612b01d210711b9c9f58cf4465636508e82d781a0630e3e59"
---

## Description
Governed early executable experiments for architecture assumptions

Problem: dependency semantics were accepted at the design abstraction level and challenged much later. H2 needs evidence for restored-key denial, original transaction outcome, external-effect uncertainty and recovery fencing; unit tests or API docs alone cannot establish these. No experiment lane exists (docs/consistency-layer-proposal.md section 7 leaves experiments attested or experimental).

Proposed fix: a lightweight experiment lane integrated with architecture adoption and the existing checker/evidence contracts. An experiment binds one critical assumption (from the readiness inventory), a disposable environment description, exact subject and dependency versions, explicit expected observations and negative controls, and a retained evidence disposition. Experiments may inform a source revision but never count as production GREEN, locked RED completion or milestone acceptance. Support recording "not runnable without owner authority" without fabricating proof or provisioning resources. Reuse existing safety/custody mechanisms (processscope, runtime closure) rather than a new test runner unless a concrete missing capability is shown.

Acceptance criteria:
1. A local synthetic database plus independent key-service experiment reproduces successful decryption after restoring a deleted wrapped-key row; a corrected denial mechanism survives that restore and its negative controls.
2. Missing or stale experiment evidence blocks only the declared dependent adoption or handoff.
3. Fake adapters cannot be labelled native proof.
4. Registering an experiment implies no cloud purchase, secret, privilege or destructive cleanup; "not runnable without owner authority" is a first-class recorded state.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
