# Legacy surface ledger reference

Gm proves every entity in the legacy MODEL is disposed; nothing proves the legacy model captured
the legacy SYSTEM. The ledger anchors design coverage to the system's mechanically enumerable
surface. It is independent of `migration.yaml` by design: a clean-break run that drops the
migration machinery keeps its completeness anchor.

```text
design/legacy/surface.yaml          # the capability disposition ledger; activates Gs-surface
```

Two named sweeps author it. **Opening sweep (Phase 0/1):** enumerate the surface mechanically
(route tables, command registrations, schema catalogs, cron and worker lists, outbound calls);
most rows start `deferred`; the opening ledger is the interrogation's work list. **Closing sweep
(after Gate 4):** re-mine the legacy system against the finished design and settle every row;
whatever the docs-first pass missed surfaces as a row that cannot be honestly disposed.

Strict root shape (unknown keys fail):

```yaml
surface_version: 1
system: <one line naming the legacy system and its shape>
classes:
  routes:                            # network API surface
    source: <where the enumeration came from>
    items:
      - name: "POST /contacts"
        disposition: covered         # covered | dropped | deferred
        via: action                  # entity | action | component | machine (covered only)
        target: Contact.create       # binds against the TARGET design (covered only)
      - name: "GET /admin/metrics"
        disposition: dropped
        rationale: <why the capability does not carry>   # dropped/deferred only
  commands: {none: <reason>}         # CLI surface; waive absent classes with a reason
  tables: ...                        # tables, collections, node labels, file stores
  jobs: ...                          # scheduled and background work
  events: ...                        # async topics consumed or produced
  integrations: ...                  # outbound external dependencies
```

## Coverage rules

- All six classes must appear, each as an inventory (`source` + non-empty `items`) or a waiver
  (`none: <reason>`), never both. A forgotten class is a missing key, which is an error.
- Every item has a `name` (unique within its class) and exactly one disposition.
- `covered` requires `via` + `target`, and the binding must resolve against the TARGET design,
  never the legacy model (the legacy model is itself under suspicion of being incomplete):
  - `entity` -> an entity in `design/domain.modelith.yaml`
  - `action` -> `Entity.action` in that model
  - `component` -> a `design/workspace.dsl` element name
  - `machine` -> `design/machines/<target>.machine.json`
- `dropped` and `deferred` require `rationale` and forbid `via`/`target`: a capability with a
  design element to point at is covered.
- A ledger of only waivers is an error; a legacy system with no surface is not a legacy system.

## Staging

Entity and action bindings need the Phase 1 model; `component` needs Phase 2's `workspace.dsl`;
`machine` needs Phase 3. A covered row whose binding artifact does not exist yet is an ordinary
staging error (like Gm's narrative bridges): author early, bind via entity/action or defer, and
settle rows as phases land.

## The gate

`Gs-surface` (`machinery check <design> --gate gs`) activates automatically when the file exists.
Deterministic: strict schema, all six classes addressed, every item disposed exactly once with
required fields, covered bindings resolve, names unique. The `checked:` line prints per-class item
counts plus covered/dropped/deferred/waived totals. LLM-attested: the enumeration is complete per
class, the waivers are true, and at Gate 4 every `deferred` rationale is deliberate, never an
opening placeholder; review the deferred count for exactly that.

## Modes

Required for rebuild, hybrid, and greenfield-with-corpus (the migration machinery may be dropped;
the completeness anchor must not be). Recommended for brownfield: add `gs` to the day-one staged
gate list alongside g2,g4. Not applicable to pure greenfield.

The repository's expanded guide is `docs/surface-ledger.md`; the worked ledger is
`examples/surreal-crm/design/legacy/surface.yaml`.
