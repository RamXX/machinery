// Conformance probe: asks the real xstate package what a machine does.
// stdin: {"package": dir, "machines": [{"id", "json", "guards", "delays", "cases": [{"id", "source", "event", "guard"}]}]}
// stdout: one JSON object per line: {"machine", "initial"} or {"machine", "case", "state", "actions"},
// either with "error" instead when XState throws.
'use strict';
const fs = require('fs');

const input = JSON.parse(fs.readFileSync(0, 'utf8'));
const xs = require(input.package);

function firstLine(e) {
  return String((e && e.message) || e).split('\n')[0].trim();
}

function leafPath(value) {
  const parts = [];
  let v = value;
  while (v && typeof v === 'object') {
    const keys = Object.keys(v);
    if (keys.length === 0) break;
    parts.push(keys[0]);
    v = v[keys[0]];
  }
  if (typeof v === 'string') parts.push(v);
  return parts.join('.');
}

function stateValue(path) {
  const segs = path.split('.');
  let v = segs[segs.length - 1];
  for (let i = segs.length - 2; i >= 0; i--) v = { [segs[i]]: v };
  return v;
}

function named(actions) {
  const out = [];
  for (const a of actions) {
    const t = a && a.type;
    if (typeof t !== 'string' || t.startsWith('xstate.') || out.includes(t)) continue;
    out.push(t);
  }
  return out;
}

function implementations(m, truthy) {
  const guards = {};
  for (const g of m.guards) guards[g] = () => g === truthy;
  return { guards, delays: m.delays };
}

const emit = (o) => process.stdout.write(JSON.stringify(o) + '\n');

for (const m of input.machines) {
  let machine;
  try {
    machine = xs.createMachine(JSON.parse(m.json));
    const [s0] = xs.initialTransition(machine.provide(implementations(m, null)));
    emit({ machine: m.id, initial: leafPath(s0.value) });
  } catch (e) {
    emit({ machine: m.id, error: firstLine(e) });
    continue;
  }
  for (const c of m.cases) {
    try {
      const mm = machine.provide(implementations(m, c.guard));
      const snap = mm.resolveState({ value: stateValue(c.source) });
      const [next, actions] = xs.transition(mm, snap, { type: c.event });
      emit({ machine: m.id, case: c.id, state: leafPath(next.value), actions: named(actions) });
    } catch (e) {
      emit({ machine: m.id, case: c.id, error: firstLine(e) });
    }
  }
}
