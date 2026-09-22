# Compile-time design reads and the Gr-reads gate

An implementation can parse a design artifact as build input. A compiler macro may read a machine
matrix, generated table, or policy vocabulary and intentionally reject tokens it does not know.
That dependency is stronger than an import and different from a test citation: changing the design
file can break compilation even when G4-import and Gt-tests are both green.

`Gr-reads` makes that dependency explicit in the Architecture Contract and binds it to repository
history. The declaration is design-side evidence. The implementation path is supplied with
`--impl`, as it is for G4 and Gt.

## Declaration grammar

`reads` is an optional non-empty list at the root of the contract v2 YAML fence, beside
`boundaries`, `externals`, `ignore`, and `dependency_rules`:

```yaml
contract_version: 2
boundaries:
  - id: policy
    code: ["lib/policy/**"]
reads:
  - artifact: machines/Principal.matrix.md
    reader: lib/policy/principal_reader.ex
    reviewed: "4fb62cc557686b64334f623c82d554ab15d50bd8"
    _comment: the compile-time macro rejects unknown policy tokens
dependency_rules:
  allow: []
  deny: []
```

Each row is a closed mapping with these fields:

| Field | Rule |
|---|---|
| `artifact` | Required portable path relative to the design root. It must resolve to a regular file. |
| `reader` | Required portable path relative to `--impl`. It must resolve to a regular file when an implementation is supplied. |
| `reviewed` | Required full 40-character lowercase Git commit. Both paths must be tracked at this commit, and it must be an ancestor of the current HEAD. |
| `_comment` | Optional non-empty string. |

Unknown keys, duplicate artifact-reader pairs, empty lists, non-portable paths, missing files, short
commit ids, and unreadable history are errors. Quote `reviewed`; an all-digit object name must not be
parsed as a YAML number.

This lower-case YAML `reads:` declaration is unrelated to matrix `READS{field, ...}` groups.
`READS{...}` declares which event payload fields one consumer uses. `reads:` declares that an
implementation file parses a design file at compile or build time. Neither discharges the other.

## Gate behavior

Without `--impl`, every declaration produces a warning:

```text
machines/Principal.matrix.md: an implementation reads this file through lib/policy/principal_reader.ex; land any edit with its reader follow-up and run with --impl to verify the review history
```

The warning is deliberate. A design-only lane cannot prove what happened in the implementation,
but it can keep the coupling visible at the commit boundary.

With `--impl`, Gr walks every commit after `reviewed` through HEAD. Any commit that changes the
artifact must change the declared reader in that same commit. The same rule applies to the current
uncommitted change set. An artifact-only change is an ERROR. A paired change is recorded in the
gate's checked count. The gate does not execute the compiler or decide whether the reader update is
correct; it proves that the implementation follow-up was present for review.

The recorded review commit is a baseline, not a content hash. History remains meaningful after
later commits: every artifact edit since that review is checked, rather than only the last diff.
Re-record the baseline only after reviewing both the artifact and reader at the new commit.
