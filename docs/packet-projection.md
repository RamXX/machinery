# Per-slice packet projection and the Gw-packet gate

A manifest-mode design can outgrow the context window of the agent that has to build it. H2's
M1 is the reference case: each of its six slices binds to exactly one BUILD shard, and root plus
shard plus ARCHITECTURE.md is 300K to 470K tokens per slice against a 200K executor budget. The
owner ruled that the sources are not split by hand and the executor is not swapped for a larger
one: machinery projects, per slice, only what the slice cites.

`machinery packet` is that projection. It reads one authored artifact, `design/slices.yaml`,
resolves every citation against the committed design, copies the cited excerpts verbatim under
their stable id and source `path:line`, and writes one packet per slice. `Gw-packet` is the
gate that holds the slice map to the design: every citation resolves, every packet fits its
declared budget, and every obligation the milestone owes is claimed by exactly one slice or
carries a recorded waiver. The projection never reads prose. A slice is bound by ids, and only
by ids.

## The authored input: `design/slices.yaml`

```yaml
schema: 1
milestones:
  - id: M1
    slices:
      - id: M1-S1
        shard: BUILD/core.md
        budget: 600000
        cites:
          - section:BUILD/core.md#8.5
          - section:BUILD/core.md#9
          - milestone:BUILD/core.md
          - section:BUILD.md#11
          - TROLE-020ea7
          - TROLE-1b5494
          - oracleset:machines/AiClientCall.oracle.md
          - matrix:Tenant
          - boundary:platform.core
          - external:external.keycloak
          - rule:platform.core -> external.keycloak
          - row:ARCHITECTURE.md#3#keycloak
          - row:ARCHITECTURE.md#6#platform.gateway -> platform.core
          - invariant:tenant-scoped-data
          - file:content/errors.yaml
      - id: M1-S2
        shard: BUILD/trust.md
        budget: 600000
        cites:
          - ...
    waivers:
      - id: AMND-45e3ab
        reason: the AgentMandate chain moved to M6 with the aggregate (owner ruling 2026-08-28)
```

`schema` is the literal integer `1`. `milestones` lists every milestone that has slices; a
milestone the build plan does not declare is an error, and a declared milestone with no entry
here owes nothing to this gate.

### Slice fields

| field | rule |
|---|---|
| `id` | `<milestone>-S<n>`, with `<milestone>` the enclosing milestone's id and `<n>` a positive integer. Unique across the file. This is the stable slice id the packet file is named by. |
| `shard` | Exactly one design-relative path to a regular `.md` file. In a manifest design this is the slice's BUILD shard (`BUILD/core.md`); in a full-mode design it may be `BUILD.md`. Every `section:` and `milestone:` citation of a file under `BUILD/` must name this shard: a slice bound to one shard never pulls another shard's content. |
| `budget` | Positive integer, bytes. The packet file must not exceed it. See "The budget metric". |
| `cites` | Non-empty list of citations (below). Order is the author's; the packet groups them by kind and keeps authored order within a kind. A citation listed twice is an error. |

### Waivers

A waiver records that an obligation the milestone owes is deliberately claimed by no slice.
`id` must be an obligation of that milestone (a waiver on an id the milestone does not owe is
stale and fails), `reason` is required, and a waived id that some slice also claims is a
contradiction and fails.

## Citations

Every citation resolves to one excerpt with one source location. Resolution is exact and
whole-token; nothing is fuzzy, nothing is inferred from prose.

| citation | resolves to | excerpt |
|---|---|---|
| `TAG-hhhhhh` (a bare stable id) or `T-TAG-NN` (a bare test id) | one row of a committed oracle table: `machines/*.oracle.md`, `formal/Policy.oracle.md`, `formal/Isolation.oracle.md` | the table header plus that row, verbatim, at `path:line` of the row |
| `oracleset:<path>` | every row of one oracle file, the same paths `ORACLESET{...}` accepts | the whole transitions table |
| `matrix:<Machine>` | `machines/<Machine>.matrix.md` | the whole file |
| `section:<path>#<id>` | one heading of a Markdown file in the design, and everything under it until the next heading of the same or a higher level | the heading and its subtree, verbatim |
| `milestone:<path>` | the slice's milestone block in one plan-bearing document: in root `BUILD.md` the block `Gb-plan` parses, from the `**M<n> - <title>**` marker to the next marker or heading; in a shard's own Build plan section the block opening with a list item `- **M<n>** (...)`, to the next such item or heading | the block, verbatim |
| `file:<path>` | one regular text file in the design | the whole file |
| `boundary:<id>` | one `boundaries[]` item of the Architecture Contract fence in `ARCHITECTURE.md` | the item's YAML lines |
| `external:<id>` | one `externals[]` item of the contract | the item's YAML lines |
| `rule:<src> -> <dst>` | one entry of `dependency_rules.allow`, `.deny`, or `.baseline` | that line, with the list it belongs to |
| `row:<path>#<section id>#<key>` | one row of a Markdown table under a section: the row whose first cell is `<key>` (the first backticked span of the cell, or the whole cell when it has none) | the table header plus that row |
| `invariant:<id>` | one `invariants[]` entry of `domain.modelith.yaml` | the entry's YAML lines, followed by its row of the root `BUILD.md` traceability matrix (the table whose header names an invariant id column) when one exists |

**Section ids.** A heading has two ids: its leading number token when the heading text starts
with one (`### 8.5 Named-unit test plan` has id `8.5`; a trailing period is dropped) and its full
text after the hashes. A `section:` citation must match exactly one heading in the file; zero
or several matches fail with the candidates named. Fenced code blocks never contribute
headings.

**Row keys.** `row:ARCHITECTURE.md#3#keycloak` finds the dependency mitigation row whose first
cell begins with `` `keycloak` ``. `row:ARCHITECTURE.md#6#platform.gateway -> platform.core`
finds the interface contract row keyed by that edge. `row:BUILD.md#6#tenant-scoped-data`
finds a traceability matrix row directly. Every table in the section's subtree is searched;
zero or several matching rows fail.

**Oracle ids.** A citation of a test id and a citation of its stable id name the same row, and
the packet carries the row once under its stable id. Test ids renumber; cite the stable id.

## What a packet contains, in order

1. **Header.** The slice id, milestone, shard, the generator stamp, and the rule that every
   excerpt is a verbatim copy whose authority is the source it names.
2. **Milestone.** The milestone's marker line, its `Demo:` and `Status:` lines when present,
   each with `path:line`, and the obligation ledger: every oracle id the milestone owes, with
   the slice that claims it or the waiver that releases it. The ledger is computed; the DoD
   prose is not copied. A living plan's DoD block carries narrative that is not an obligation,
   and copying it would bind the packet to prose. A slice that wants its milestone's plan text
   cites `milestone:<path>` explicitly.
3. **Sections and files.** Every `section:`, `milestone:`, and `file:` excerpt.
4. **Oracle rows.** Every cited row under its stable id with its table header, and every
   `oracleset:` table whole.
5. **Matrices.** Every `matrix:` file.
6. **Architecture Contract.** Every `boundary:`, `external:`, `rule:`, and `row:` excerpt.
7. **Invariants.** Every `invariant:` excerpt with its traceability row.
8. **Acceptance entry shape.** A `design/acceptance/M<n>.yaml` skeleton whose `dod_ids` list is
   the full obligation set of the milestone, annotated with the claiming slice, so the executor
   sees what the milestone review will bind.
9. **Sources.** Every design file the packet drew from, with the line ranges it drew. The packet
   binds to the excerpted bytes it carries and to the slice map, whose SHA-256 the header
   states; it does not bind to whole files, so an edit elsewhere in a source never changes it.

Every excerpt is headed by `### <citation> (<path>:<first>-<last>)`, with the line range
1-based and inclusive, followed by the source lines copied byte for byte. Markdown excerpts are
delimited by `<!-- begin <citation> -->` and `<!-- end <citation> -->` markers; YAML and table
excerpts are fenced.

## Determinism

The packet is a pure function of the design bytes and `slices.yaml`. Two runs of the same
machinery version over the same bytes produce byte-identical packets. The only stamp is the
`machinery-version:` line every generated artifact carries, so a packet regenerated by a
newer release differs in that line and, when the projection rules changed, in what the
release notes say.

Excerpts carry their source line numbers, so an edit above a cited excerpt in the same file
moves the reference and changes the packet. That is a change of the citation's address, not a
prose dependency: re-wording prose in place, or removing prose below every cited excerpt,
leaves every packet byte-identical, which is what the no-prose-parsing rule promises.

## The budget metric

The budget is bytes of the packet file, and the gate is on bytes only. The byte count is
deterministic, needs no tokenizer, no vendored vocabulary, and no network, which is the
standard every other machinery gate meets. The token estimate the size report prints beside
the byte count uses one fixed divisor:

```
PacketBytesPerToken = 3
```

Declare a budget as `executor tokens * 3`: a 200K-token executor gets `budget: 600000`.
Measured tokenizers sit near 3.5 bytes per token on this kind of Markdown (H2's own M1
measurement: 1.13 MB of root plus shard at 324K tokens), so a packet that fits its byte
budget fits the executor with margin. The divisor is a documented constant, not a knob:
changing it is a release-notes change, never a per-design setting.

## The gate: `Gw-packet`

Activates when `design/slices.yaml` exists; `--gate gw` forces it, and forcing it without the
file is an error. Runs after `Gb-plan`, whose milestone parse it reuses, and before
`Ge-embed`. Fails closed:

- **The slice map parses and agrees with the design.** Every milestone named is one the build
  plan declares. Every slice id has the required shape, is unique, and belongs to its
  milestone. Every shard is a regular file in the design. Every citation resolves to exactly
  one excerpt.
- **Every packet fits its budget.** The projected bytes for each slice are compared with the
  declared budget. An over-budget slice is an ERROR naming the bytes and the budget.
- **Every obligation is claimed exactly once.** The obligations a milestone owes are the
  committed oracle ids its DoD cites whole-token, `ORACLESET{...}` expanded, the same set
  `Ga-accept` binds acceptance evidence to. Each must be claimed by exactly one slice (a bare
  id or an `oracleset:` that contains it) or carry a waiver. Unclaimed, double-claimed, and
  contradicted (claimed and waived) obligations are ERRORs.

The `checked:` line reports milestones, slices, obligations claimed and waived, and the
projected bytes per slice.

`machinery packet` runs the same gate before writing anything and writes nothing when it
fails. A packet that would have been over budget, or a projection that would have dropped an
obligation, is never handed to an executor.

## The command

```
machinery packet <design-dir> --milestone M1 [--slice M1-S3] --out <dir>
```

Writes `<dir>/<slice-id>.packet.md` for the selected slice, or for every slice of the
milestone when `--slice` is absent, and prints one size line per packet:

```
packet M1-S1 -> out/M1-S1.packet.md: 412336 bytes of 600000 budget (137446 token-equivalents at 3 bytes per token)
```

The gate runs over the whole milestone even when one slice is selected, because the
coverage rule is a property of the milestone, not of a slice.

## Authoring the slice map

The one manual step is writing `slices.yaml` from the plan that already assigns work to
slices in prose. A workable order:

1. List the milestone's obligations: `machinery check <design> --gate ga` reports them as the
   DoD-cited ids, or read `dod_ids` of the milestone's acceptance entry when one exists.
2. Give every obligation to exactly one slice as a bare stable id, or waive it with the
   ruling that moved it.
3. For each slice, cite the shard sections its executor must read (`section:`), the machine
   contracts behind its rows (`matrix:`), the contract items and mitigation rows it must
   honor (`boundary:`, `rule:`, `row:`), and the invariants it enforces (`invariant:`).
4. Run `machinery packet` and read the size line. A slice over budget cites too much for one
   executor: narrow the sections to subsections, or split the slice and move claims.
5. Commit `slices.yaml`. `Gw-packet` now holds it on every `machinery check`.

Packets are outputs, never inputs: they are regenerated for every executor run and are not
committed to the design.
